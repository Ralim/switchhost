package nacp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	cnmt "github.com/ralim/switchhost/formats/CNMT"
	istorage "github.com/ralim/switchhost/formats/IStorage"
	nca "github.com/ralim/switchhost/formats/NCA"
	partitionfs "github.com/ralim/switchhost/formats/partitionFS"
	"github.com/ralim/switchhost/formats/utils"
	"github.com/ralim/switchhost/keystore"
	"github.com/ralim/switchhost/settings"
)

//https://switchbrew.org/wiki/NACP_Format
type Language int

const (
	AmericanEnglish      Language = 0
	BritishEnglish       Language = 1
	Japanese             Language = 2
	French               Language = 3
	German               Language = 4
	LatinAmericanSpanish Language = 5
	Spanish              Language = 6
	Italian              Language = 7
	Dutch                Language = 8
	CanadianFrench       Language = 9
	Portuguese           Language = 10
	Russian              Language = 11
	Korean               Language = 12
	TraditionalChinese   Language = 13
	SimplifiedChinese    Language = 14
)

type NacpTitleEntry struct {
	Language Language
	Title    string
}

type NACP struct {
	Titles                 map[Language]NacpTitleEntry
	DisplayVersion         string
	SupportedLanguageFlags uint32
}

func ExtractNACP(keystore *keystore.Keystore, cnmt *cnmt.ContentMetaAttributes, file io.ReaderAt, securePartition *partitionfs.PartionFS, securePartitionOffset uint64) (*NACP, error) {
	if control, ok := cnmt.Contents["Control"]; ok {
		controlNca := securePartition.GetByName(control.ID)
		if controlNca == nil {
			return nil, fmt.Errorf("unable to find control.nacp by id %v", control.ID)
		}

		NCAMetaHeader, err := nca.ParseNCAEncryptedHeader(keystore, file, securePartitionOffset+controlNca.StartOffset)
		if err != nil {
			return nil, fmt.Errorf("parsing NCA encrypted header failed with - %w", err)
		}
		fsHeader, err := nca.GetFSHeader(NCAMetaHeader, 0)
		if err != nil {
			return nil, err
		}

		section, err := nca.DecryptMetaNCADataSection(keystore, file, NCAMetaHeader, securePartitionOffset+controlNca.StartOffset)
		if err != nil {
			return nil, err
		}
		if fsHeader.FSType == 0 {
			romFsHeader, err := istorage.ReadHeader(section)
			if err != nil {
				return nil, err
			}
			fEntries, err := istorage.ReadFileEntries(section, *romFsHeader)
			if err != nil {
				return nil, err
			}

			if entry, ok := fEntries["control.nacp"]; ok {
				nacp, err := ReadNACP(section, *romFsHeader, entry)
				if err != nil {
					return nil, err
				}
				return &nacp, nil
			}
		} else {
			return nil, errors.New("unsupported type " + control.ID)
		}

	}
	return nil, errors.New("no control.nacp found")
}

func ReadNACP(data []byte, romFsHeader istorage.Header, fileEntry istorage.FileEntry) (NACP, error) {
	offset := romFsHeader.DataOffset + fileEntry.Offset
	titles := map[Language]NacpTitleEntry{}
	for i := 0; i < 16; i++ {
		appTitleBytes := data[offset+(uint64(i)*0x300) : offset+(uint64(i)*0x300)+0x200]
		nameBytes := utils.CString(appTitleBytes)
		titles[Language(i)] = NacpTitleEntry{Language: Language(i), Title: string(nameBytes)}
	}

	displayVersion := utils.CString(data[offset+0x3060 : offset+0x3060+0x10])
	supportedLanguageFlags := binary.BigEndian.Uint32(data[offset+0x302C : offset+0x302C+0x4])

	return NACP{Titles: titles, DisplayVersion: string(displayVersion), SupportedLanguageFlags: supportedLanguageFlags}, nil

}

func (n *NACP) GetSuggestedTitle(settings *settings.Settings) string {
	// Return the titles in preferred order
	// If not fall back by Language order
	for index := range settings.PreferredLangOrder {
		v, ok := n.Titles[Language(index)]
		if ok {
			if len(v.Title) > 0 {
				return v.Title
			}
		}
	}

	for _, v := range n.Titles {
		if len(v.Title) > 0 {
			return v.Title
		}
	}
	return ""
}

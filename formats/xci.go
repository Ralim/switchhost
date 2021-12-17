package formats

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	cnmt "github.com/ralim/switchhost/formats/CNMT"
	nacp "github.com/ralim/switchhost/formats/NACP"
	nca "github.com/ralim/switchhost/formats/NCA"
	partitionfs "github.com/ralim/switchhost/formats/partitionFS"
	"github.com/ralim/switchhost/keystore"
	"github.com/ralim/switchhost/settings"
	"github.com/rs/zerolog/log"
)

const (
	XCIHeaderSize              = 0x200
	XCIHeaderMagicStringOffset = 0x100
	XCIRootPartionHeaderOffset = 0x130
)

func ParseXCIToMetaData(keystore *keystore.Keystore, settings *settings.Settings, reader io.ReaderAt) (FileInfo, error) {
	info := FileInfo{}
	header := make([]byte, XCIHeaderSize)
	_, err := reader.ReadAt(header, 0)
	if err != nil {
		return info, err
	}
	XCIHeaderString := string(header[XCIHeaderMagicStringOffset : XCIHeaderMagicStringOffset+4])
	if XCIHeaderString != "HEAD" {
		return info, fmt.Errorf("invalid XCI headerBytes. Expected 'HEAD', got >%s<", XCIHeaderString)
	}

	rootPartitionOffset := binary.LittleEndian.Uint64(header[XCIRootPartionHeaderOffset : XCIRootPartionHeaderOffset+8])

	rootHfs0, err := partitionfs.ReadSection(reader, int64(rootPartitionOffset))
	if err != nil {
		return info, fmt.Errorf("reading XCI PartionFS failed with - %w", err)
	}

	secureHfs0, secureOffset, err := readSecurePartition(reader, rootHfs0, rootPartitionOffset)
	if err != nil {
		return info, err
	}

	for _, pfs0File := range secureHfs0.FileEntryTable {

		fileOffset := secureOffset + int64(pfs0File.StartOffset)

		if strings.Contains(pfs0File.Name, "cnmt.nca") {

			NCAMetaHeader, err := nca.ParseNCAEncryptedHeader(keystore, reader, uint64(fileOffset))
			if err != nil {
				return info, fmt.Errorf("ParseNCAEncryptedHeader failed with - %w", err)
			}
			section, err := nca.DecryptMetaNCADataSection(keystore, reader, NCAMetaHeader, uint64(fileOffset))
			if err != nil {
				return info, fmt.Errorf("DecryptMetaNCADataSection failed with - %w", err)
			}
			currpfs0, err := partitionfs.ReadSection(bytes.NewReader(section), 0x0)
			if err != nil {
				return info, fmt.Errorf("ReadSection failed with - %w", err)
			}

			currCnmt, err := cnmt.ParseBinary(currpfs0, section)
			if err != nil {
				return info, fmt.Errorf("ParseBinary failed with - %w", err)
			}

			if currCnmt.Type != cnmt.DLC {
				nacp, err := nacp.ExtractNACP(keystore, currCnmt, reader, secureHfs0, uint64(secureOffset))
				if err != nil {
					log.Warn().Msgf("Failed to extract NACP info from file %+v - %v", &currCnmt.Type, err.Error())
				} else {
					// currCnmt.Ncap = nacp
					info.EmbeddedTitle = nacp.GetSuggestedTitle(settings)
				}
			}
			//Update the info
			info.TitleID = currCnmt.TitleId
			info.Version = currCnmt.Version
			info.Type = currCnmt.Type

		}
	}
	return info, nil
}

func readSecurePartition(file io.ReaderAt, hfs0 *partitionfs.PartionFS, rootPartitionOffset uint64) (*partitionfs.PartionFS, int64, error) {
	for _, hfs0File := range hfs0.FileEntryTable {
		offset := int64(rootPartitionOffset) + int64(hfs0File.StartOffset)

		if hfs0File.Name == "secure" {
			securePartition, err := partitionfs.ReadSection(file, offset)
			if err != nil {
				return nil, 0, err
			}
			return securePartition, offset, nil
		}
	}
	return nil, 0, nil
}

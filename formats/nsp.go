package formats

import (
	"bytes"
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

//Implements the minimum to parse the details we care about out of an NSP file

func ParseNSPToMetaData(keystore *keystore.Keystore, settings *settings.Settings, reader io.ReaderAt) (FileInfo, error) {
	info := FileInfo{}
	pfs0Header, err := partitionfs.ReadSection(reader, 0)
	if err != nil {
		return info, fmt.Errorf("reading NSP PartionFS failed with - %w", err)
	}

	for _, pfs0File := range pfs0Header.FileEntryTable {

		if strings.HasSuffix(pfs0File.Name, "cnmt.nca") {
			NCAMetaHeader, err := nca.ParseNCAEncryptedHeader(keystore, reader, pfs0File.StartOffset)
			if err != nil {
				return info, fmt.Errorf("ParseNCAEncryptedHeader failed with - %w", err)
			}
			section, err := nca.DecryptMetaNCADataSection(keystore, reader, NCAMetaHeader, pfs0File.StartOffset)
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
				nacp, err := nacp.ExtractNACP(keystore, currCnmt, reader, pfs0Header, 0)
				if err != nil {
					log.Warn().Int("type", int(currCnmt.Type)).Err(err).Msg("Failed to extract NACP info from file")
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

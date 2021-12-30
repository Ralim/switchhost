package formats

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	cnmt "github.com/ralim/switchhost/formats/CNMT"
	nca "github.com/ralim/switchhost/formats/NCA"
	partitionfs "github.com/ralim/switchhost/formats/partitionFS"
	"github.com/ralim/switchhost/keystore"
	"github.com/ralim/switchhost/settings"
	"github.com/rs/zerolog/log"
)

// Validations, trying to sanity check our files are intact

// Find the CNMT section in the file, as this holds the content metadaata, then inside this, has hashes
// Once these are found validate these against the file

func ValidateNSPHash(keystore *keystore.Keystore, settings *settings.Settings, reader ReaderRequired) error {
	pfs0Header, err := partitionfs.ReadSection(reader, 0)
	if err != nil {
		return fmt.Errorf("reading NSP PartionFS failed with - %w", err)
	}
	var fileCNMT *cnmt.ContentMetaAttributes
	fileCNMT = nil
	for _, pfs0File := range pfs0Header.FileEntryTable {

		if strings.HasSuffix(pfs0File.Name, "cnmt.nca") {
			NCAMetaHeader, err := nca.ParseNCAEncryptedHeader(keystore, reader, pfs0File.StartOffset)
			if err != nil {
				return fmt.Errorf("ParseNCAEncryptedHeader failed with - %w", err)
			}
			section, err := nca.DecryptMetaNCADataSection(keystore, reader, NCAMetaHeader, pfs0File.StartOffset)
			if err != nil {
				return fmt.Errorf("DecryptMetaNCADataSection failed with - %w", err)
			}
			currpfs0, err := partitionfs.ReadSection(bytes.NewReader(section), 0x0)
			if err != nil {
				return fmt.Errorf("ReadSection failed with - %w", err)
			}
			currCnmt, err := cnmt.ParseBinary(currpfs0, section)
			if err != nil {
				return fmt.Errorf("ParseBinary failed with - %w", err)
			}
			fileCNMT = currCnmt
		}
	}
	for _, pfs0File := range pfs0Header.FileEntryTable {

		if strings.HasSuffix(pfs0File.Name, ".nca") && !strings.HasSuffix(pfs0File.Name, "cnmt.nca") {
			//This is a data partition, look to match it against one of the hashes, and if it matches then check its checksum

			hasher := sha256.New()
			reader.Seek(int64(pfs0File.StartOffset), io.SeekStart)
			if _, err := io.CopyN(hasher, reader, int64(pfs0File.Size)); err != nil {
				return err
			}
			partitionHash := hasher.Sum(nil)

			validated := false
			for _, c := range fileCNMT.Contents {
				if strings.HasPrefix(pfs0File.Name, c.ID) {
					matchingHash := c
					// Read out the partition

					if !bytes.Equal(partitionHash, matchingHash.Hash) {
						fmt.Println("Bang")
						return errors.New("hash failed validation")
					}
					log.Info().Str("part", pfs0File.Name).Msg("validated correctly")
					validated = true
				}
			}
			if !validated {
				return fmt.Errorf("partition >%s< could not be validated as no hash in CNMT", pfs0File.Name)
			}

		} else if strings.HasSuffix(pfs0File.Name, ".ncz") {
			//Compressed partition, need to handle decompression

			hasher := sha256.New()
			reader.Seek(int64(pfs0File.StartOffset), io.SeekStart)
			uncompressedheaderLength := 0x4000
			if pfs0File.Size < uint64(uncompressedheaderLength) {
				uncompressedheaderLength = pfs0Header.Size
			}
			if _, err := io.CopyN(hasher, reader, int64(uncompressedheaderLength)); err != nil {
				return err
			}

			// compresedAreaLength := pfs0File.Size - uint64(uncompressedheaderLength)
			if pfs0File.Size > uint64(uncompressedheaderLength) {
				//Use zstandard to decompress the rest of the file
				magic := make([]byte, 8)
				_, err := reader.Read(magic)
				if err != nil {
					return err
				}
				if !bytes.Equal(magic, []byte("NCZSECTN")) {
					return fmt.Errorf("failed to validate partition >%s<, bad NCZ >NCZSECTN< header >%v<", pfs0File.Name, string(magic))
				}
				_, err = reader.Read(magic)
				if err != nil {
					return err
				}
				sectionCount := binary.LittleEndian.Uint64(magic)
				//Read out the section headers

			}

			partitionHash := hasher.Sum(nil)

			validated := false
			for _, c := range fileCNMT.Contents {
				if strings.HasPrefix(pfs0File.Name, c.ID) {
					matchingHash := c
					// Read out the partition

					if !bytes.Equal(partitionHash, matchingHash.Hash) {
						fmt.Println("Bang")
						return errors.New("hash failed validation")
					}
					log.Info().Str("part", pfs0File.Name).Msg("validated correctly")
					validated = true
				}
			}
			if !validated {
				return fmt.Errorf("partition >%s< could not be validated as no hash in CNMT", pfs0File.Name)
			}
		}
	}

	return nil
}

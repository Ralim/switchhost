package formats

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/klauspost/compress/zstd"
	aesctr "github.com/ralim/switchhost/formats/AESCTR"
	cnmt "github.com/ralim/switchhost/formats/CNMT"
	nca "github.com/ralim/switchhost/formats/NCA"
	nsz "github.com/ralim/switchhost/formats/NSZ"
	partitionfs "github.com/ralim/switchhost/formats/partitionFS"
	"github.com/ralim/switchhost/keystore"
	"github.com/ralim/switchhost/settings"
	"github.com/rs/zerolog/log"
)

const UNCOMPRESSABLE_HEADER_SIZE int64 = 0x4000

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
		if err := validatePFS0File(pfs0File, reader, fileCNMT); err != nil {
			return err
		}
	}

	return nil
}

func validatePFS0File(pfs0File partitionfs.FileEntryTableItem, reader ReaderRequired, fileCNMT *cnmt.ContentMetaAttributes) error {

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
		uncompressedheaderLength := UNCOMPRESSABLE_HEADER_SIZE
		if pfs0File.Size < uint64(uncompressedheaderLength) {
			uncompressedheaderLength = int64(pfs0File.Size)
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
			sectionCount := int64(binary.LittleEndian.Uint64(magic))
			sections := make([]nsz.NSZSection, sectionCount)
			//Read out the section headers
			for i := 0; i < int(sectionCount); i++ {
				sect, err := nsz.NSZSectionFromReader(reader)
				if err != nil {
					return err
				}
				sections[i] = *sect
			}

			if (sections[0].Offset - UNCOMPRESSABLE_HEADER_SIZE) > 0 {
				section := nsz.NSZSectionDummy(UNCOMPRESSABLE_HEADER_SIZE, sections[0].Offset-UNCOMPRESSABLE_HEADER_SIZE)
				sect := []nsz.NSZSection{section}
				sections = append(sect, sections...)
			}

			_, err = reader.Read(magic)
			if err != nil {
				return err
			}
			//Step back after reading magic
			_, err = reader.Seek(-8, io.SeekCurrent)
			if err != nil {
				return err
			}
			useBlockDecompressor := false
			if bytes.Equal(magic, []byte("NCZBLOCK")) {
				useBlockDecompressor = true
			}
			var decompressingReader io.Reader
			if useBlockDecompressor {
				blockDecompressor, err := nsz.NewBlockDecompressor(reader)

				if err != nil {
					return err
				}
				decompressingReader = blockDecompressor
			} else {

				zstdReader, err := zstd.NewReader(reader)
				if err != nil {
					return err
				}

				decompressingReader = zstdReader
			}
			for sectNum, section := range sections {
				// Chain varies by crypto type
				// If crypto type is 3 or 4, then we want to do: file -> zstandard -> crypto -> hash
				// else  then we want to do                    : file -> zstandard -> hash
				offset := section.Offset
				var prehashReader io.Reader
				// Now we either chain this into crypto or the hash directly
				if section.CryptoType == 3 || section.CryptoType == 4 {
					cipherStream, err := aesctr.NewAESCTREncrypter(decompressingReader, section.CryptoKey, section.CryptoCounter, []byte{})
					if err != nil {
						return err
					}
					//On section 0, account for the jump over the uncompressed first chunk
					if sectNum == 0 {
						uncompressedSize := int64(uncompressedheaderLength) - section.Offset
						if uncompressedSize > 0 {
							offset += uncompressedSize
						}
					}
					cipherStream.Seek(uint64(offset))
					prehashReader = cipherStream

				} else {
					prehashReader = decompressingReader
				}
				//Now we can copy all the bytes into the hasher

				_, err = io.CopyN(hasher, prehashReader, section.Size-(offset-section.Offset))
				if err != nil {
					return err
				}

			}
			partitionHash := hasher.Sum(nil)

			validated := false
			for _, c := range fileCNMT.Contents {
				if strings.HasPrefix(pfs0File.Name, c.ID) {
					matchingHash := c

					if !bytes.Equal(partitionHash, matchingHash.Hash) {
						return fmt.Errorf("hash failed validation %X != %X", partitionHash, matchingHash.Hash)
					}
					log.Debug().Str("part", pfs0File.Name).Msg("validated correctly")
					validated = true
				}
			}
			if !validated {
				return fmt.Errorf("partition >%s< could not be validated as no hash in CNMT", pfs0File.Name)
			}
		}

		partitionHash := hasher.Sum(nil)

		validated := false
		for _, c := range fileCNMT.Contents {
			if strings.HasPrefix(pfs0File.Name, c.ID) {
				matchingHash := c
				// Read out the partition

				if !bytes.Equal(partitionHash, matchingHash.Hash) {
					return errors.New("hash failed validation")
				}
				log.Debug().Str("part", pfs0File.Name).Msg("validated correctly")
				validated = true
			}
		}
		if !validated {
			return fmt.Errorf("partition >%s< could not be validated as no hash in CNMT", pfs0File.Name)
		}
	}
	return nil
}

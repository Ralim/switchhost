package nca

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/ralim/switchhost/formats/xts"
	"github.com/ralim/switchhost/keystore"
)

//https://switchbrew.org/wiki/NCA

const (
	// Sections
	NCASectionCode = 0
	NCASectionData = 1
	NCASectionLogo = 2
	// Content types
	NCAContentProgram    = 0
	NCAContentMeta       = 1
	NCAContentControl    = 2
	NCAContentManual     = 3
	NCAContentData       = 4
	NCAContentPublicData = 5
	// Format consts
	NCAHeaderLength = 0xC00
	NCASectorSize   = 0x200
)

type NCAHeader struct {
	HeaderBytes    []byte // Raw Decrypted header
	RightsID       []byte // rights id [ 0x10]
	ProgramID      uint64 // programID of the file
	Distribution   byte
	ContentType    byte
	KeyGeneration2 byte
	KeyGeneration1 byte
	EncryptedKeys  []byte // 4 * 0x10
	CryptoType     byte
}

func DecryptMetaNCADataSection(keystore *keystore.Keystore, reader io.ReaderAt, header *NCAHeader, ncaOffset uint64) ([]byte, error) {

	dataSectionIndex := 0

	fsHeader, err := GetFSHeader(header, dataSectionIndex)
	if err != nil {
		return nil, err
	}

	entry := GetFSEntry(header, dataSectionIndex)

	if entry.Size == 0 {
		return nil, errors.New("empty section")
	}

	encodedEntryContent := make([]byte, entry.Size)
	entryOffset := int64(ncaOffset) + int64(entry.StartOffset)
	_, err = reader.ReadAt(encodedEntryContent, entryOffset)
	if err != nil {
		return nil, fmt.Errorf("decryptingMetaNCA Failed during reading encoded entry with - %w", err)
	}
	if fsHeader.EncType != 3 {
		return nil, errors.New("non supported encryption type [encryption type:" + string(fsHeader.EncType))
	}

	decoded, err := decryptAesCtr(keystore, header, fsHeader, entry.StartOffset, entry.Size, encodedEntryContent)
	if err != nil {
		return nil, err
	}
	hashInfo, err := fsHeader.getHashInfo()
	if err != nil {
		return nil, err
	}

	return decoded[hashInfo.PFS0HeaderOffset:], nil
}

func ParseNCAEncryptedHeader(keystore *keystore.Keystore, reader io.ReaderAt, ncaOffset uint64) (*NCAHeader, error) {
	// Validate we have the required header key for parsing
	headerKey, err := keystore.GetHeaderKey()
	if err != nil {
		return nil, errors.New("cant decode NCA data without `header_key`")
	}
	encryptedNCADataBlock := make([]byte, NCAHeaderLength)
	_, err = reader.ReadAt(encryptedNCADataBlock, int64(ncaOffset))

	if err != nil {
		return nil, fmt.Errorf("reading NCA header raised %w ", err)
	}

	ncaHeader, err := decryptNcaHeader(headerKey, encryptedNCADataBlock)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt NCA header - %w", err)
	}
	return ncaHeader, nil

}

func decryptNcaHeader(headerKey, encHeader []byte) (*NCAHeader, error) {

	c, err := xts.NewCipher(aes.NewCipher, headerKey)

	if err != nil {
		return nil, fmt.Errorf("cipher could not be created - %w", err)
	}

	length := 0x400
	decryptedHeader, err := decryptNCAHeaderBlock(c, encHeader, length, NCASectorSize, 0)
	if err != nil {
		return nil, fmt.Errorf("inital NCA(1/2) decryption failed - %w", err)
	}

	magic := string(decryptedHeader[NCASectorSize : NCASectorSize+0x04])
	if !strings.HasPrefix(magic, "NCA") {
		return nil, fmt.Errorf("NCA decryption failed, invalid magic - %s", magic)
	}

	if magic == "NCA3" {
		length = 0xC00
		decryptedHeader, err = decryptNCAHeaderBlock(c, encHeader, length, NCASectorSize, 0)
		if err != nil {
			return nil, fmt.Errorf("secondary NCA(3) decryption failed - %w", err)
		}
	}

	result := NCAHeader{HeaderBytes: decryptedHeader}

	result.Distribution = decryptedHeader[0x204]
	result.ContentType = decryptedHeader[0x205]
	result.RightsID = decryptedHeader[0x230 : 0x230+0x10]

	result.ProgramID = binary.LittleEndian.Uint64(decryptedHeader[0x210 : 0x210+0x8])
	result.KeyGeneration1 = decryptedHeader[0x206]
	result.CryptoType = decryptedHeader[0x207]
	result.KeyGeneration2 = decryptedHeader[0x220]

	encryptedKeysAreaOffset := 0x300
	result.EncryptedKeys = decryptedHeader[encryptedKeysAreaOffset : encryptedKeysAreaOffset+(0x10*4)]

	return &result, nil
}

func decryptNCAHeaderBlock(c *xts.Cipher, header []byte, length, sectorSize, sectorNum int) ([]byte, error) {
	decrypted := make([]byte, len(header))
	for pos := 0; pos < length; pos += sectorSize {
		pos := sectorSize * sectorNum
		c.Decrypt(decrypted[pos:pos+sectorSize], header[pos:pos+sectorSize], uint64(sectorNum))
		sectorNum++
	}
	return decrypted, nil
}

func (n *NCAHeader) getKeyRevision() int {
	keyGeneration := int(math.Max(float64(n.KeyGeneration1), float64(n.KeyGeneration2)))
	keyRevision := keyGeneration - 1
	if keyGeneration == 0 {
		return 0
	}
	return int(keyRevision)
}

func decryptAesCtr(keystore *keystore.Keystore, ncaHeader *NCAHeader, fsHeader *FSHeader, offset uint32, size uint32, encoded []byte) ([]byte, error) {
	keyRevision := ncaHeader.getKeyRevision()
	cryptoType := ncaHeader.CryptoType

	if cryptoType != 0 {
		return nil, errors.New("unsupported crypto type")
	}
	key, err := keystore.GetAppKey(uint8(keyRevision))
	if err != nil {
		return nil, fmt.Errorf("missing key - %02x -> %w", keyRevision, err)
	}

	decKey, err := decryptAes128Ecb(ncaHeader.EncryptedKeys[0x20:0x30], key)
	if err != nil {
		return nil, fmt.Errorf("ECB error - %w", err)
	}
	counter := make([]byte, 0x10)
	binary.BigEndian.PutUint64(counter, uint64(fsHeader.Generation))
	binary.BigEndian.PutUint64(counter[8:], uint64(offset/0x10))

	c, _ := aes.NewCipher(decKey)

	decContent := make([]byte, size)

	s := cipher.NewCTR(c, counter)
	s.XORKeyStream(decContent, encoded[0:size])

	return decContent, nil
}

func decryptAes128Ecb(data, key []byte) ([]byte, error) {

	cipher, _ := aes.NewCipher([]byte(key))
	decrypted := make([]byte, len(data))
	size := 16
	if len(data)%size != 0 {
		return decrypted, errors.New("invalid input length")
	}

	for bs, be := 0, size; bs < len(data); bs, be = bs+size, be+size {
		cipher.Decrypt(decrypted[bs:be], data[bs:be])
	}

	return decrypted, nil
}

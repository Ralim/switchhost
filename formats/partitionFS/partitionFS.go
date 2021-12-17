package partitionfs

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/ralim/switchhost/formats/utils"
)

//https://wiki.oatmealdome.me/PFS0_(File_Format)
//https://switchbrew.org/wiki/XCI
//https://switchbrew.org/wiki/NCA (bottom)
// The PFS0 (Partition FS) format is used within NCAs, and is a Switch-exclusive format.
// Fairly simple file header structure

const (
	PFS0HeaderLength      = 0x0C
	PFSfileEntryTableSize = 0x18
	HFSfileEntryTableSize = 0x40
	PFSStaticHeaderLength = 0x10 // Length up until the dynamic content
	PFS0Magic             = "PFS0"
	HFS0Magic             = "HFS0"
)

type FileEntryTableItem struct {
	StartOffset uint64
	Size        uint64
	Name        string
}

// PartionFS struct is the parsed representation of the file header PFS0/HFS0 section
type PartionFS struct {
	Size           int
	HeaderLen      int // Length of the entire header up until file contents start
	FileEntryTable []FileEntryTableItem
}

func ReadSection(reader io.ReaderAt, offset int64) (*PartionFS, error) {

	header := make([]byte, PFS0HeaderLength)
	_, err := reader.ReadAt(header, offset)
	if err != nil {
		return nil, fmt.Errorf("reading the PFS0 header failed with %w", err)
	}
	headerMagic := string(header[0:0x04])
	fileEntryTableSize := 0
	switch headerMagic {
	case PFS0Magic:
		fileEntryTableSize = PFSfileEntryTableSize
	case HFS0Magic:
		fileEntryTableSize = HFSfileEntryTableSize
	default:
		return nil, fmt.Errorf("invalid filesystem magic. Wanted %s/%s, got >%s<", PFS0Magic, HFS0Magic, headerMagic)
	}

	pfs := &PartionFS{}

	fileCount := int(binary.LittleEndian.Uint16(header[0x4:0x8]))
	stringTableLength := int(binary.LittleEndian.Uint16(header[0x8:0xC]))

	pfs.HeaderLen = PFSStaticHeaderLength + (fileEntryTableSize * fileCount) + stringTableLength
	// Now read in the entire header
	headerRemainderBuffer := make([]byte, pfs.HeaderLen-PFSStaticHeaderLength)

	_, err = reader.ReadAt(headerRemainderBuffer, offset+PFSStaticHeaderLength)
	if err != nil {
		return nil, fmt.Errorf("reading the PFS0 FileEntryTable+Strings failed with %w", err)
	}

	fileTable, err := parseFileEntryTableAndStrings(fileCount, pfs.HeaderLen, fileEntryTableSize, headerRemainderBuffer)
	if err != nil {
		return nil, fmt.Errorf("parsing the PFS0 FileEntryTable+Strings failed with %w", err)
	}
	pfs.FileEntryTable = fileTable
	return pfs, nil
}

func parseFileEntryTableAndStrings(fileCount, headerLength, fileEntryTableSize int, data []byte) ([]FileEntryTableItem, error) {

	files := make([]FileEntryTableItem, fileCount)
	stringTableStartsAt := fileCount * fileEntryTableSize
	for i := 0; i < fileCount; i++ {
		//Parse the file info details
		recordStart := fileEntryTableSize * i
		files[i].StartOffset = binary.LittleEndian.Uint64(data[recordStart:recordStart+0x08]) + uint64(headerLength)
		files[i].Size = binary.LittleEndian.Uint64(data[recordStart+0x08 : recordStart+0x10])
		stringOffset := binary.LittleEndian.Uint32(data[recordStart+0x10 : recordStart+0x14])
		//here after is either padding (PFS0) or more checksum info (HFS0)
		// For now we dont care
		stringStart := stringTableStartsAt + int(stringOffset)
		if stringStart >= len(data) {
			return files, fmt.Errorf("corrupted File Table Entry, decoded string length beyond end of header for entry %d, gave %d", i, stringStart)
		}
		files[i].Name = utils.CString(data[stringStart:])
	}
	return files, nil
}

func (partition *PartionFS) GetByName(id string) *FileEntryTableItem {
	for _, fileEntry := range partition.FileEntryTable {
		if strings.Contains(fileEntry.Name, id) {
			return &fileEntry
		}
	}
	return nil
}

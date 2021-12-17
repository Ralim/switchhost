package istorage

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type Header struct {
	HeaderSize          uint64
	DirHashTableOffset  uint64
	DirHashTableSize    uint64
	DirMetaTableOffset  uint64
	DirMetaTableSize    uint64
	FileHashTableOffset uint64
	FileHashTableSize   uint64
	FileMetaTableOffset uint64
	FileMetaTableSize   uint64
	DataOffset          uint64
}

type FileEntry struct {
	Parent    uint32
	Sibling   uint32
	Offset    uint64
	Size      uint64
	Hash      uint32
	Name      string
	Name_size uint32
}

const (
	FileTableEntrySize = 0x20
)

// ReadHeader will parse the header data section into the header struct
func ReadHeader(data []byte) (*Header, error) {
	if len(data) < 8*10 {
		return nil, fmt.Errorf("IStorage length too short %d", len(data))
	}
	// Read out the array of uint64 values
	values := make([]uint64, 10)
	for i := 0; i < 10; i++ {
		values[i] = binary.LittleEndian.Uint64(data[(8 * i):(8 * (i + 1))])
	}
	header := &Header{
		HeaderSize:          values[0],
		DirHashTableOffset:  values[1],
		DirHashTableSize:    values[2],
		DirMetaTableOffset:  values[3],
		DirMetaTableSize:    values[4],
		FileHashTableOffset: values[5],
		FileHashTableSize:   values[6],
		FileMetaTableOffset: values[7],
		FileMetaTableSize:   values[8],
		DataOffset:          values[9],
	}

	return header, nil
}

// ReadFileEntries Will return all of the Fileentry records contained in the data
func ReadFileEntries(data []byte, header Header) (map[string]FileEntry, error) {
	if header.FileMetaTableOffset+header.FileMetaTableSize > uint64(len(data)) {
		return nil, errors.New("data too small / bad header")
	}
	dirBytes := data[header.FileMetaTableOffset : header.FileMetaTableOffset+header.FileMetaTableSize]
	result := map[string]FileEntry{}

	offset := uint32(0x0)
	for offset < uint32(header.FileHashTableSize) {
		entry := FileEntry{}
		entry.Parent = binary.LittleEndian.Uint32(dirBytes[offset : offset+0x4])
		entry.Sibling = binary.LittleEndian.Uint32(dirBytes[offset+0x4 : offset+0x8])
		entry.Offset = binary.LittleEndian.Uint64(dirBytes[offset+0x8 : offset+0x10])
		entry.Size = binary.LittleEndian.Uint64(dirBytes[offset+0x10 : offset+0x18])
		entry.Hash = binary.LittleEndian.Uint32(dirBytes[offset+0x18 : offset+0x1C])
		entry.Name_size = binary.LittleEndian.Uint32(dirBytes[offset+0x1C : offset+0x20])
		entry.Name = string(dirBytes[offset+FileTableEntrySize : (offset+FileTableEntrySize)+entry.Name_size])
		result[entry.Name] = entry
		offset = offset + FileTableEntrySize + entry.Name_size
	}
	return result, nil
}

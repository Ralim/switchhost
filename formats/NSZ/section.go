package nsz

import (
	"encoding/binary"
	"io"
)

type NSZSection struct {
	Offset        int64
	Size          int64
	CryptoType    int64
	Pad           int64
	CryptoKey     []byte
	CryptoCounter []byte
}

func NSZSectionFromReader(reader io.Reader) (*NSZSection, error) {
	headerSize := (4 * 8) + (16 * 2)
	data := make([]byte, headerSize)
	if _, err := reader.Read(data); err != nil {
		return nil, err
	}
	nz := &NSZSection{}
	nz.Offset = int64(binary.LittleEndian.Uint64(data[0:8]))
	nz.Size = int64(binary.LittleEndian.Uint64(data[8:16]))
	nz.CryptoType = int64(binary.LittleEndian.Uint64(data[16:24]))
	nz.Pad = int64(binary.LittleEndian.Uint64(data[24:32]))
	nz.CryptoKey = data[32:48]
	nz.CryptoCounter = data[48:64]
	return nz, nil
}

func NSZSectionDummy(size, offset int64) NSZSection {
	return NSZSection{
		Size:       size,
		Offset:     offset,
		CryptoType: 0,
	}
}

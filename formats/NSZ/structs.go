package nsz

import (
	"encoding/binary"
	"io"
)

type BlockHeader struct {
	Magic                   []byte
	Version                 int8
	Type                    int8
	Unused                  int8
	BlockSizeExponent       int8
	NumberOfBlocks          int32
	DecompressedSize        int64
	CompressedBlockSizeList []int32
}

func NewBlockHeader(reader io.Reader) (*BlockHeader, error) {
	// self.numberOfBlocks = f.readInt32()
	// self.decompressedSize = f.readInt64()
	// self.compressedBlockSizeList = [f.readInt32() for _ in range(self.numberOfBlocks)]
	header := &BlockHeader{
		Magic: make([]byte, 8),
	}
	if _, err := reader.Read(header.Magic); err != nil {
		return nil, err
	}
	scratch := make([]byte, 8)
	if _, err := reader.Read(scratch); err != nil {
		return nil, err
	}
	header.Version = int8(scratch[0])
	header.Type = int8(scratch[1])
	header.Unused = int8(scratch[2])
	header.BlockSizeExponent = int8(scratch[3])

	header.NumberOfBlocks = int32(binary.LittleEndian.Uint32(scratch[4:8]))

	if _, err := reader.Read(scratch); err != nil {
		return nil, err
	}
	header.DecompressedSize = int64(binary.LittleEndian.Uint64(scratch))
	header.CompressedBlockSizeList = make([]int32, header.NumberOfBlocks)
	for i := 0; i < int(header.NumberOfBlocks); i++ {
		scratch2 := make([]byte, 4)
		if _, err := reader.Read(scratch2); err != nil {
			return nil, err
		}
		header.CompressedBlockSizeList[i] = int32(binary.LittleEndian.Uint32(scratch2))
	}
	return header, nil

}

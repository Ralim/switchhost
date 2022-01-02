package nsz

import (
	"io"
	"math"

	"github.com/klauspost/compress/zstd"
)

type blockInfo struct {
	fileoffset            int64 // Position this block starts at in the source file
	blockCompressedLength int64 // How many bytes can be read from the compressed source at a maximum (before it runs into the next block)
}

func (b *blockInfo) GetReader(reader io.ReadSeeker) io.Reader {
	return io.LimitReader(reader, b.blockCompressedLength)
}

type Decompressor struct {
	io.Reader
	source io.ReadSeeker

	initalOffset      int64
	header            *BlockHeader
	blockSize         int64
	compressionBlocks []blockInfo

	currentVirtualPos          int64
	currentDecompressor        *zstd.Decoder
	currentNotCompressedReader io.Reader
	trace                      int64
}

func NewBlockDecompressor(reader io.ReadSeeker) (*Decompressor, error) {
	dec := &Decompressor{
		source:            reader,
		currentVirtualPos: 0,
	}
	header, err := NewBlockHeader(reader)
	if err != nil {
		return nil, err
	}
	dec.header = header
	n, err := reader.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	dec.initalOffset = n
	//Now need to convert all of the recorded lengths into the index of blocks
	//Each block represents 2^BlockSizeExponent
	dec.blockSize = int64(math.Pow(2, float64(header.BlockSizeExponent)))
	dec.compressionBlocks = make([]blockInfo, len(header.CompressedBlockSizeList))
	for i, bs := range header.CompressedBlockSizeList {
		dec.compressionBlocks[i] = blockInfo{
			fileoffset:            n,
			blockCompressedLength: int64(bs),
		}

		n += int64(bs)
	}
	return dec, nil
}

func (d *Decompressor) Read(p []byte) (n int, err error) {
	//read out from the existing zstd compressor if it exists
	n = 0
	if d.currentDecompressor != nil || d.currentNotCompressedReader != nil {
		//Try and read as much as we can from this existing compressor
		if d.currentDecompressor != nil {
			n, err = d.currentDecompressor.Read(p)
		} else {
			n, err = d.currentNotCompressedReader.Read(p)
		}
		//Need to check if we hit EOF, so we can close the decompressor
		isEOF := err == io.EOF
		d.currentVirtualPos += int64(n)
		d.trace += int64(n)
		if isEOF {
			if d.currentDecompressor != nil {
				d.currentDecompressor.Close()
			}

			d.currentDecompressor = nil
			d.currentNotCompressedReader = nil
			err = nil
		}
		return n, err
	}
	//Load in the next decompression block if we can
	nextBlock := d.currentVirtualPos / d.blockSize
	if int(nextBlock) >= len(d.compressionBlocks) || d.currentVirtualPos > d.header.DecompressedSize {
		return 0, io.EOF
	}
	nextBlocko := d.compressionBlocks[nextBlock]
	//Seek to appropriate starting point
	{
		_, err = d.source.Seek(nextBlocko.fileoffset, io.SeekStart)
		if err != nil {
			return 0, err
		}
	}
	//Decide if we are reading a compressed block or an uncompressed (skipped) block
	expectedDecompressedBlockSize := int64(d.blockSize)

	if int(nextBlock) == len(d.compressionBlocks)-1 {
		expectedDecompressedBlockSize = int64(d.header.DecompressedSize - d.currentVirtualPos)
	}
	//If expectedDecompressedBlockSize is the same as the recorded block size; its not compressed
	if int64(expectedDecompressedBlockSize) == nextBlocko.blockCompressedLength {
		d.currentNotCompressedReader = nextBlocko.GetReader(d.source)
		d.currentDecompressor = nil
	} else {
		zstdReader, err := zstd.NewReader(nextBlocko.GetReader(d.source))
		if err != nil {
			return 0, err
		}
		d.currentDecompressor = zstdReader
		d.currentNotCompressedReader = nil
	}
	//Try and read as much as we can from this existing compressor
	if d.currentDecompressor != nil {
		n, err = d.currentDecompressor.Read(p)
	} else {
		n, err = d.currentNotCompressedReader.Read(p)
	}
	//Need to check if we hit EOF, so we can close the decompressor
	if err == io.EOF {
		if d.currentDecompressor != nil {
			d.currentDecompressor.Close()
		}

		d.currentDecompressor = nil
		d.currentNotCompressedReader = nil
	}
	d.currentVirtualPos += int64(n)
	d.trace = int64(n)
	return n, err

}

func (d *Decompressor) Close() {
	if d.currentDecompressor != nil {
		d.currentDecompressor.Close()
	}
}

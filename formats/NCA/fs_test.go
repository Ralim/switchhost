package nca

import (
	"testing"
)

func TestGetFsEntry(t *testing.T) {
	t.Parallel()
	header := &Header{
		HeaderBytes: make([]byte, 0x240+0x10),
	}
	//Append in the data that is read out
	//start offset
	header.HeaderBytes = append(header.HeaderBytes, []byte{0xAA, 0x55, 0x01, 0x00}...)
	//end offset
	header.HeaderBytes = append(header.HeaderBytes, []byte{0xAA, 0x55, 0x02, 0x00}...)
	entry := GetFSEntry(header, 1)
	if entry.Size != 0x2000000 {
		t.Errorf("expected a size of 0x2000000, got 0x%x", entry.Size)
	}
	if entry.StartOffset != 0x000155AA*0x200 {
		t.Errorf("expected a StartOffset of 0x000155AA* 0x200, got 0x%x", entry.StartOffset)
	}
	if entry.EndOffset != 0x000255AA*0x200 {
		t.Errorf("expected a EndOffset of 0x000155AA* 0x200, got 0x%x", entry.EndOffset)
	}
}

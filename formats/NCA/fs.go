package nca

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

type FSHeader struct {
	EncType       byte //(0 = Auto, 1 = None, 2 = AesCtrOld, 3 = AesCtr, 4 = AesCtrEx)
	FSType        byte //(0 = RomFs, 1 = PartitionFs)
	HashType      byte // (0 = Auto, 2 = HierarchicalSha256, 3 = HierarchicalIntegrity (Ivfc))
	FSHeaderBytes []byte
	Generation    uint32
}

type FSEntry struct {
	StartOffset uint32
	EndOffset   uint32
	Size        uint32
}

type HashInfo struct {
	PFS0HeaderOffset uint64
	PFS0size         uint64
}

const (
	PFS0HeaderOffset = 0x280
	PFS0HashSize     = 0x20
	PFS0EntryOffset  = 0x240
	PFS0EntrySize    = 0x10
)

func GetFSEntry(ncaHeader *Header, index int) FSEntry {
	fsEntryOffset := PFS0EntryOffset + PFS0EntrySize*index
	fsEntryBytes := ncaHeader.HeaderBytes[fsEntryOffset : fsEntryOffset+PFS0EntrySize]

	entryStartOffset := binary.LittleEndian.Uint32(fsEntryBytes[0x0:0x4]) * 0x200
	entryEndOffset := binary.LittleEndian.Uint32(fsEntryBytes[0x4:0x8]) * 0x200

	return FSEntry{StartOffset: entryStartOffset, EndOffset: entryEndOffset, Size: entryEndOffset - entryStartOffset}
}

func GetFSHeader(ncaHeader *Header, index int) (*FSHeader, error) {
	fsHeaderHashOffset := PFS0HeaderOffset + PFS0HashSize*index
	fsHeaderHash := ncaHeader.HeaderBytes[fsHeaderHashOffset : fsHeaderHashOffset+0x20]

	fsHeaderOffset := 0x400 + 0x200*index
	fsHeaderBytes := ncaHeader.HeaderBytes[fsHeaderOffset : fsHeaderOffset+0x200]

	actualHash := sha256.Sum256(fsHeaderBytes)

	if !bytes.Equal(actualHash[:], fsHeaderHash) {
		return nil, fmt.Errorf("fs headerBytes hash mismatch -> %v <-> %v", actualHash, fsHeaderHash)
	}

	result := FSHeader{FSHeaderBytes: fsHeaderBytes}

	result.FSType = fsHeaderBytes[0x2:0x3][0]
	result.HashType = fsHeaderBytes[0x3:0x4][0]
	result.EncType = fsHeaderBytes[0x4:0x5][0]

	generationBytes := fsHeaderBytes[0x140 : 0x140+0x4] //generation
	result.Generation = binary.LittleEndian.Uint32(generationBytes)

	return &result, nil
}

func (fh *FSHeader) getHashInfo() (*HashInfo, error) {
	hashInfoBytes := fh.FSHeaderBytes[0x8:0x100]
	result := HashInfo{}
	if fh.HashType == 2 {

		result.PFS0HeaderOffset = binary.LittleEndian.Uint64(hashInfoBytes[0x38 : 0x38+0x8])
		result.PFS0size = binary.LittleEndian.Uint64(hashInfoBytes[0x40 : 0x40+0x8])
		return &result, nil
	} else if fh.HashType == 3 {
		result.PFS0HeaderOffset = binary.LittleEndian.Uint64(hashInfoBytes[0x88 : 0x88+0x8])
		result.PFS0size = binary.LittleEndian.Uint64(hashInfoBytes[0x90 : 0x90+0x8])
		return &result, nil
	}
	return nil, errors.New("non supported hash type")
}

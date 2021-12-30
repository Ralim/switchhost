package formats

import (
	"io"

	cnmt "github.com/ralim/switchhost/formats/CNMT"
)

type FileType uint8

// FileInfo is the parsed metadata around a file
// This contains all the data used by the rest of the package
type FileInfo struct {
	Name string

	TitleID       uint64
	Version       uint32
	EmbeddedTitle string
	Type          cnmt.MetaType
	Size          int64
}

type ReaderRequired interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

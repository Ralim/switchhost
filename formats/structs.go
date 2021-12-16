package formats

import cnmt "github.com/ralim/switchhost/formats/CNMT"

type FileType uint8

type FileInfo struct {
	Name string

	TitleID       uint64
	Version       uint32
	EmbeddedTitle string
	Type          cnmt.MetaType
	Size          int64
}

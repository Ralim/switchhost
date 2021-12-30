package cnmt

import (
	"encoding/binary"
	"errors"
	"fmt"

	partitionfs "github.com/ralim/switchhost/formats/partitionFS"
)

// CNMT -> https://switchbrew.org/wiki/CNMT
// A.K.A PackagedContentMeta

const (
	ContentMetaType_SystemProgram        = 1
	ContentMetaType_SystemData           = 2
	ContentMetaType_SystemUpdate         = 3
	ContentMetaType_BootImagePackage     = 4
	ContentMetaType_BootImagePackageSafe = 5
	ContentMetaType_Application          = 0x80
	ContentMetaType_Patch                = 0x81
	ContentMetaType_AddOnContent         = 0x82
	ContentMetaType_Delta                = 0x83
)

type ContentType int

// MetaType is the main type of the contents, so base game, updates, dlc etc
type MetaType int

const (
	Unknown  MetaType = 0
	BaseGame MetaType = 1
	Update   MetaType = 2
	DLC      MetaType = 3
)

type Content struct {
	Type string
	ID   string
	Size uint64
	Hash []byte
}

type ContentMetaAttributes struct {
	TitleId  uint64
	Version  uint32
	Type     MetaType
	Contents map[string]Content
}

type ContentMeta struct {
	Text                          string
	Type                          string
	ID                            string
	Version                       int
	RequiredDownloadSystemVersion string
	Content                       []Content
	Digest                        string
	KeyGenerationMin              string
	RequiredSystemVersion         string
	OriginalId                    string
}

func (t *MetaType) String() string {
	switch *t {
	case Unknown:
		return "Unknown"
	case BaseGame:
		return "Base"
	case Update:
		return "Update"
	case DLC:
		return "DLC"
	default:
		return "Unknown"
	}
}

func ParseBinary(pfs0 *partitionfs.PartionFS, data []byte) (*ContentMetaAttributes, error) {
	if pfs0 == nil || len(pfs0.FileEntryTable) != 1 {
		return nil, errors.New("invalid PartionFS")
	}
	cnmtFile := pfs0.FileEntryTable[0]
	cnmt := data[int64(cnmtFile.StartOffset):]
	titleId := binary.LittleEndian.Uint64(cnmt[0:0x8])
	version := binary.LittleEndian.Uint32(cnmt[0x8:0xC])
	tableOffset := binary.LittleEndian.Uint16(cnmt[0xE:0x10])
	contentEntryCount := binary.LittleEndian.Uint16(cnmt[0x10:0x12])
	contents := map[string]Content{}
	for i := uint16(0); i < contentEntryCount; i++ {
		position := 0x20 /*size of cnmt header*/ + tableOffset + (i * uint16(0x38))
		hashData := cnmt[position : position+0x20]
		ncaId := cnmt[position+0x20 : position+0x20+0x10]
		sizeData := cnmt[position+0x30 : position+0x30+0x06]
		// only 6 bytes, so need to add two zero pads
		sizeData = append([]byte{0, 0}, sizeData...)
		size := binary.LittleEndian.Uint64(sizeData)
		contentType := ""
		switch cnmt[position+0x36] {
		case 0:
			contentType = "Meta"
		case 1:
			contentType = "Program"
		case 2:
			contentType = "Data"
		case 3:
			contentType = "Control"
		case 4:
			contentType = "HtmlDocument"
		case 5:
			contentType = "LegalInformation"
		case 6:
			contentType = "DeltaFragment"
		}
		contents[contentType] = Content{ID: fmt.Sprintf("%x", ncaId), Hash: hashData, Size: size, Type: contentType}
	}
	metaType := Unknown
	switch cnmt[0xC] {
	case ContentMetaType_Application:
		metaType = BaseGame
	case ContentMetaType_AddOnContent:
		metaType = DLC
	case ContentMetaType_Patch:
		metaType = Update
	}

	return &ContentMetaAttributes{Contents: contents, Version: version, TitleId: titleId, Type: metaType}, nil
}

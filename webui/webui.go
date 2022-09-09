package webui

import (
	"errors"

	_ "embed"

	"github.com/ralim/switchhost/library"
	"github.com/ralim/switchhost/titledb"
)

//go:embed templates/index.html
var titlePageTemplate string

//go:embed templates/detail.html
var detailPageTemplate string

//go:embed templates/skeleton.min.css
var SkeletonCss []byte

var ErrBadTemplate = errors.New("bad template file")

// WebUI is a fairly dumb package that manages transforming basic titleID + titlesdb

type WebUI struct {
	lib     *library.Library
	titleDB *titledb.TitlesDB
}

func NewWebUI(lib *library.Library, titleDB *titledb.TitlesDB) *WebUI {
	return &WebUI{
		lib:     lib,
		titleDB: titleDB,
	}
}

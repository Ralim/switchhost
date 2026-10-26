package webui

import (
	"fmt"
	"io"
	"strings"
)

func (web *WebUI) RenderTitleInfo(titleID uint64, writer io.Writer) error {
	//Render out a web page of the info we have on the title
	titleDetails, ok := web.titleDB.QueryGameFromTitleID(titleID)
	if !ok {
		return ErrBadTemplate
	}
	template := detailPageTemplate

	template = strings.ReplaceAll(template, "{GameImageURI}", titleDetails.IconURL)
	template = strings.ReplaceAll(template, "{GameBannerImageURI}", titleDetails.BannerURL)
	template = strings.ReplaceAll(template, "{GameTitle}", titleDetails.Name)
	//Generate info table
	filesTracked := web.lib.FileIndex.GetAllRecordsForTitle(titleID)
	tableInfo := ""
	for _, record := range filesTracked {
		tableInfo += fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>%d</td></tr>\n", record.Name, record.Version, record.Size)
	}
	template = strings.ReplaceAll(template, "{GameDetailsTableContents}", tableInfo)
	if _, err := writer.Write([]byte(template)); err != nil {
		return ErrBadTemplate
	}
	return nil
}

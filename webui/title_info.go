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

	template = strings.Replace(template, "{GameImageURI}", titleDetails.IconURL, -1)
	template = strings.Replace(template, "{GameBannerImageURI}", titleDetails.BannerURL, -1)
	template = strings.Replace(template, "{GameTitle}", titleDetails.Name, -1)
	//Generate info table
	filesTracked := web.lib.FileIndex.GetAllRecordsForTitle(titleID)
	tableInfo := ""
	for _, record := range filesTracked {
		tableInfo += fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>%d</td></tr>\n", record.Name, record.Version, record.Size)
	}
	template = strings.Replace(template, "{GameDetailsTableContents}", tableInfo, -1)
	if _, err := writer.Write([]byte(template)); err != nil {
		return ErrBadTemplate
	}
	return nil
}

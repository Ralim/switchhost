package webui

import (
	"fmt"
	"io"
	"strings"
)

func (web *WebUI) RenderGameListing(writer io.Writer) error {
	templateParts := strings.Split(titlePageTemplate, "{GameTitleCards}")
	if len(templateParts) != 2 {
		return ErrBadTemplate
	}
	if _, err := writer.Write([]byte(templateParts[0])); err != nil {
		return ErrBadTemplate
	}

	i := 0
	for _, game := range web.lib.FileIndex.ListTitleFiles() {
		if i > 0 && i%4 == 0 {
			if _, err := writer.Write([]byte(`</div><div class="row">`)); err != nil {
				return ErrBadTemplate
			}
		}
		gameData := []byte(web.renderGameIconCard(game.TitleID))
		if len(gameData) > 0 {
			if _, err := writer.Write(gameData); err != nil {
				return ErrBadTemplate
			}
			i++
		}
	}
	if _, err := writer.Write([]byte(templateParts[1])); err != nil {
		return ErrBadTemplate
	}
	return nil
}

func (web *WebUI) renderGameIconCard(titleID uint64) string {
	titleDeets, ok := web.titleDB.QueryGameFromTitleID(titleID)
	if !ok {
		return ""
	}
	if titleDeets.IconURL != "" {

		template := `<div class="three columns">
<a href="%s%d" class="card">
  <h6>%s</h6>
  <img class="u-max-full-width" src="%s" />
</a>
</div>
`
		return fmt.Sprintf(template, "/info/", titleID, titleDeets.Name, titleDeets.IconURL)

	} else {
		// Todo, non image card
		return ""
	}
}

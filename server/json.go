package server

import (
	"encoding/json"
	"fmt"
	"io"

	titledb "github.com/ralim/switchhost/titledb"
	"github.com/ralim/switchhost/utilities"
)

// JSON handler generates a response json listing of the resources known to this instance
type fileEntry struct {
	URL  string `json:"url"`
	Size uint64 `json:"size"`
	Name string `json:"title"`
}
type jsonIndex struct {
	Files   []fileEntry                     `json:"files"`
	Folders []string                        `json:"directories"`
	MOTD    string                          `json:"success"`
	TitleDB map[string]titledb.TitleDBEntry `json:"titledb"`
}

func (server *Server) generateJSONPayload(writer io.Writer, hostNameToUse string, useHTTPS bool) error {
	response := jsonIndex{
		Files:   []fileEntry{},
		MOTD:    server.settings.ServerMOTD,
		TitleDB: make(map[string]titledb.TitleDBEntry),
	}

	for _, file := range server.library.ListFiles() {
		response.Files = append(response.Files, fileEntry{URL: server.GenerateVirtualFilePath(file, hostNameToUse, useHTTPS), Size: 1, Name: utilities.CleanName(file.Name)})
		fileinfo, ok := server.library.LookupFileInfo(file)
		if ok {
			response.TitleDB[fileinfo.StringID] = fileinfo
		}
	}
	respBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("JSON creation failed with - %w", err)
	}

	_, err = writer.Write(respBytes)
	return err
}

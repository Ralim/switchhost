package server

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ralim/switchhost/titledb"
	"github.com/ralim/switchhost/utilities"
)

// JSON handler generates a response json listing of the resources known to this instance
type fileEntry struct {
	URL  string `json:"url"`
	Size int64  `json:"size"`
	Name string `json:"title"`
}
type jsonIndex struct {
	Files           []fileEntry                     `json:"files"`
	Folders         []string                        `json:"directories"`
	MOTD            *string                         `json:"success"`
	TitleDB         map[string]titledb.TitleDBEntry `json:"titledb"`
	BackupLocations []string                        `json:"locations"`
}

func (server *Server) generateFileJSONPayload(writer io.Writer, hostNameToUse string, useHTTPS bool) error {
	response := jsonIndex{
		Files:           []fileEntry{},
		TitleDB:         make(map[string]titledb.TitleDBEntry),
		BackupLocations: server.settings.JSONLocations,
	}
	if len(server.settings.ServerMOTD) > 0 {
		response.MOTD = &server.settings.ServerMOTD
	}

	for _, file := range server.library.ListFiles() {
		response.Files = append(response.Files, fileEntry{URL: server.GenerateVirtualFilePath(file, hostNameToUse, useHTTPS), Size: file.Size, Name: utilities.CleanName(file.Name)})
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

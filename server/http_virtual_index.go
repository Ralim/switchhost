package server

import (
	"fmt"
	"github.com/ralim/switchhost/utilities"
	"net/http"
	"path"
	"strings"
)

// Virtual HTTP index
// Generates a HTTP "index page" that shows all known tracked files
// Basically a virtual directory listing, just like we do for FTP

func (server *Server) httpHandleVirtualIndex(respWriter http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(respWriter, "Only GET is allowed", http.StatusMethodNotAllowed)
		return
	}
	respWriter.Write([]byte("<!DOCTYPE html>\n<html lang=\"en-US\">\n<head></head><body><table>"))
	allTitles := server.library.FileIndex.ListTitleFiles()
	for _, file := range allTitles {
		ext := path.Ext(file.Path)
		ext = strings.ToLower(ext)
		fileFinalName := fmt.Sprintf("%s [%016X][v%d]%s", utilities.CleanName(file.Name), file.TitleID, file.Version, ext)
		base := fmt.Sprintf("/vfile/%d/%d/data.bin", file.TitleID, file.Version)

		respWriter.Write([]byte(fmt.Sprintf("<tr><td><a href=\"%s\">%s</a></td></tr>\n", base, fileFinalName)))
	}
	respWriter.Write([]byte("</table></body>"))
}

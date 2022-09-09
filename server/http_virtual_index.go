package server

import (
	"fmt"
	"github.com/ralim/switchhost/index"
	"github.com/ralim/switchhost/utilities"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
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
	head := ""
	head, req.URL.Path = ShiftPath(req.URL.Path)
	log.Info().Str("path", req.URL.Path).Str("head", head).Msg("HTTP Request")

	if head == "" {
		server.renderHTTPGameIndex(respWriter, req)
	}
	baseTitleID, err := strconv.ParseUint(head, 10, 64)
	if err != nil {
		return
	}
	if len(req.URL.Path) <= 1 {
		server.renderHTTPGameFiles(baseTitleID, respWriter)
		return
	}
	//Otherwise its a file request
	splits1 := strings.Split(req.URL.Path[1:], ".")
	splits := strings.Split(splits1[0], "-")
	log.Info().Str("path", req.URL.Path).Strs("keys", splits).Msg("HTTP Request")

	TitleID, err := strconv.ParseUint(splits[0], 10, 64)
	if err != nil {
		return
	}
	version, err := strconv.ParseUint(splits[1], 10, 64)
	if err != nil {
		return
	}
	server.serveHTTPGameFiles(TitleID, uint32(version), respWriter, req)

}
func (server *Server) serveHTTPGameFiles(titleID uint64, version uint32, respWriter http.ResponseWriter, req *http.Request) {

	log.Info().Uint64("title", titleID).Uint32("version", version).Msg("HTTP File Serving Request")
	info, ok := server.library.FileIndex.GetFileRecord(titleID, version)
	if !ok {
		return
	}
	file, err := os.Open(info.Path)
	if err != nil {
		return
	}
	finfo, err := file.Stat()
	if err != nil {
		return
	}
	size := finfo.Size()
	if err != nil {
		http.Error(respWriter, "Path not found", http.StatusNotFound)
		return
	}

	defer file.Close()
	//respWriter.Header().Add("content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
	respWriter.Header().Add("Accept-Ranges", "bytes")
	rangeHeader, ok := req.Header["Range"]
	if ok {
		startb, endb, err := server.parseRangeHeader(rangeHeader[0])
		if err != nil {
			http.Error(respWriter, "Invalid range bytes", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		_, err = file.Seek(int64(startb), io.SeekStart)
		if err != nil {
			http.Error(respWriter, "Invalid range bytes", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		//Now safe to send final headers and push the payload out
		respWriter.Header().Add("Content-Range", fmt.Sprintf("%d-%d/%d", startb, endb, size))
		respWriter.WriteHeader(http.StatusPartialContent)

		_, _ = io.CopyN(respWriter, file, endb-startb+1)
	} else {
		_, _ = io.Copy(respWriter, file)
	}
}
func (server *Server) renderHTTPGameFiles(titleID uint64, respWriter http.ResponseWriter) {
	_, _ = respWriter.Write([]byte("<!DOCTYPE HTML PUBLIC \"-//W3C//DTD HTML 3.2 Final//EN\">\n<html>\n <head>\n  <title>Index of /</title>\n </head>\n <body>\n<h1>Index of /</h1>\n<ul><ul><li><a href=\"/\"> Parent Directory</a></li>"))
	records, ok := server.library.FileIndex.GetTitleRecords(titleID)

	if ok {
		writeFile := func(w http.ResponseWriter, file index.FileOnDiskRecord, fType string) {
			ext := path.Ext(file.Path)
			ext = strings.ToLower(ext)
			fileFinalName := fmt.Sprintf("%s - %s - [%d][v%d]", utilities.CleanName(file.Name), fType, file.TitleID, file.Version)
			base := fmt.Sprintf("%d-%d%s", file.TitleID, file.Version, ext)

			_, _ = respWriter.Write([]byte(fmt.Sprintf("<li><a href=\"%s\"> %s</a></li>\n", base, fileFinalName)))
		}
		if records.BaseTitle != nil {
			writeFile(respWriter, *records.BaseTitle, "Base")
		}
		if records.Update != nil {
			writeFile(respWriter, *records.Update, "Update")
		}
		for _, file := range records.DLC {
			writeFile(respWriter, file, "DLC")

		}

	}
	_, _ = respWriter.Write([]byte("</ul>\n</body></html>"))
}
func (server *Server) renderHTTPGameIndex(respWriter http.ResponseWriter, req *http.Request) {
	_, _ = respWriter.Write([]byte("<!DOCTYPE HTML PUBLIC \"-//W3C//DTD HTML 3.2 Final//EN\">\n<html>\n <head>\n  <title>Index of /</title>\n </head>\n <body>\n<h1>Index of /</h1>\n<ul><ul><li><a href=\"/\"> Parent Directory</a></li>"))
	allTitles := server.library.FileIndex.ListTitleFiles()
	for _, file := range allTitles {
		fileFinalName := fmt.Sprintf("%s [%016X]", utilities.CleanName(file.Name), file.TitleID)
		base := fmt.Sprintf("%d/", file.TitleID)

		_, _ = respWriter.Write([]byte(fmt.Sprintf("<li><a href=\"%s\"> %s/</a></li>\n", base, fileFinalName)))
	}
	_, _ = respWriter.Write([]byte("</ul>\n</body></html>"))
}

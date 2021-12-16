package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/ralim/switchhost/webui"
)

var ErrInvalidHeader = errors.New("invalid request header")

func (server *Server) StartHTTP() {

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", server.settings.HTTPPort), server))
}

func (server *Server) httpHandleJSON(respWriter http.ResponseWriter, r *http.Request) {
	fmt.Printf("Serving JSON to %+v,%v\n", r.Proto, r.Host)
	respWriter.Header().Set("Content-Type", "application/json")
	err := server.generateFileJSONPayload(respWriter, r.Host, false)
	if err != nil {
		fmt.Println(err)
		http.Error(respWriter, "Generating index failed", http.StatusInternalServerError)
		return
	}
}
func (server *Server) httpHandleTitlesDB(respWriter http.ResponseWriter, r *http.Request) {
	respWriter.Header().Set("Content-Type", "application/json")
	err := server.titledb.DumpToJSON(respWriter)
	if err != nil {
		fmt.Println(err)
		http.Error(respWriter, "Generating index failed", http.StatusInternalServerError)
		return
	}
}
func (server *Server) parseRangeHeader(rangeHeader string) (int64, int64, error) {
	rangeHeader = strings.ReplaceAll(rangeHeader, "bytes=", "")
	rangeSplit := strings.Split(rangeHeader, "-")
	if len(rangeSplit) != 2 {
		return 0, 0, ErrInvalidHeader

	}
	startB, err := strconv.ParseInt(rangeSplit[0], 10, 64)
	if err != nil {
		return 0, 0, ErrInvalidHeader
	}
	endB, err := strconv.ParseInt(rangeSplit[1], 10, 64)
	if err != nil {
		return 0, 0, ErrInvalidHeader
	}
	return startB, endB, nil
}
func (server *Server) httpHandlevFile(respWriter http.ResponseWriter, r *http.Request) {
	fmt.Printf("Serving File %+v to %+v\n", r.URL, r.Header)

	reader, name, size, err := server.getFileFromVirtualPath(r.URL.Path)
	if err != nil {
		fmt.Println(err)
		http.Error(respWriter, "Path not found", http.StatusNotFound)
		return
	}

	defer reader.Close()
	respWriter.Header().Add("content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
	respWriter.Header().Add("Accept-Ranges", "bytes")
	rangeHeader, ok := r.Header["Range"]
	if ok {
		startb, endb, err := server.parseRangeHeader(rangeHeader[0])
		if err != nil {
			fmt.Println(err)
			http.Error(respWriter, "Invalid range bytes", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		_, err = reader.Seek(int64(startb), io.SeekStart)
		if err != nil {
			fmt.Println(err)
			http.Error(respWriter, "Invalid range bytes", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		//Now safe to send final headers and push the payload out
		respWriter.Header().Add("Content-Range", fmt.Sprintf("%d-%d/%d", startb, endb, size))
		respWriter.WriteHeader(http.StatusPartialContent)

		_, err = io.CopyN(respWriter, reader, endb-startb+1)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		_, err = io.Copy(respWriter, reader)
		if err != nil {
			fmt.Println(err)
			http.Error(respWriter, "Sending file failed", http.StatusInternalServerError)
			return
		}
	}

}
func (server *Server) httpHandleIndex(respWriter http.ResponseWriter, r *http.Request) {
	respWriter.Header().Set("Content-Type", "text/html; charset=UTF-8")
	err := server.webui.RenderGameListing(respWriter)

	if err != nil {
		fmt.Println(err)
		http.Error(respWriter, "Sending file failed", http.StatusInternalServerError)
		return
	}
}
func (server *Server) httpHandleCSS(respWriter http.ResponseWriter, r *http.Request) {
	respWriter.Header().Set("Content-Type", "text/css")
	_, err := respWriter.Write(webui.SkeletonCss)
	if err != nil {
		fmt.Println(err)
		http.Error(respWriter, "Sending file failed", http.StatusInternalServerError)
		return
	}
}
func (server *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(res, "Only GET and PUT are allowed", http.StatusMethodNotAllowed)

	}

	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	switch head {
	case "vfile":
		server.httpHandlevFile(res, req)
	case "index.json":
		server.httpHandleJSON(res, req)
	case "titledb.json":
		server.httpHandleTitlesDB(res, req)
	case "skeleton.min.css":
		server.httpHandleCSS(res, req)
	case "index.html":
		fallthrough
	case "":
		fallthrough
	case "/":
		server.httpHandleIndex(res, req)
	}
}

//ShiftPath splits off the front portion of the provided path into head and then returns the remainder in tail
func ShiftPath(pathIn string) (head, tail string) {
	pathIn = path.Clean("/" + pathIn)
	i := strings.Index(pathIn[1:], "/") + 1
	if i <= 0 {
		return pathIn[1:], "/"
	}
	return pathIn[1:i], pathIn[i:]
}

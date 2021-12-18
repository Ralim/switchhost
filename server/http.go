package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/justinas/alice"
	"github.com/ralim/switchhost/webui"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

var ErrInvalidHeader = errors.New("invalid request header")

func (server *Server) StartHTTP() {

	c := alice.New()

	// Install the logger handler with default output on the console
	c = c.Append(hlog.NewHandler(log.Logger))

	// Install some provided extra handler to set some request's context fields.
	// Thanks to that handler, all our logs will come with some prepopulated fields.
	c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Stringer("url", r.URL).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("")
	}))
	c = c.Append(hlog.RemoteAddrHandler("ip"))
	c = c.Append(hlog.UserAgentHandler("user_agent"))
	c = c.Append(hlog.RefererHandler("referer"))

	// Here is your final handleS
	h := c.Then(server)
	http.Handle("/", h)

	if err := (http.ListenAndServe(fmt.Sprintf(":%d", server.settings.HTTPPort), nil)); err != nil {
		log.Fatal().Err(err).Msg("Startup failed")
	}
}

func (server *Server) httpHandleJSON(respWriter http.ResponseWriter, r *http.Request) {
	respWriter.Header().Set("Content-Type", "application/json")
	err := server.generateFileJSONPayload(respWriter, r.Host, false)
	if err != nil {
		http.Error(respWriter, "Generating index failed", http.StatusInternalServerError)
		return
	}
}
func (server *Server) httpHandleTitlesDB(respWriter http.ResponseWriter, r *http.Request) {
	respWriter.Header().Set("Content-Type", "application/json")
	err := server.titledb.DumpToJSON(respWriter)
	if err != nil {
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

	reader, name, size, err := server.getFileFromVirtualPath(r.URL.Path)
	if err != nil {
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
			http.Error(respWriter, "Invalid range bytes", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		_, err = reader.Seek(int64(startb), io.SeekStart)
		if err != nil {
			http.Error(respWriter, "Invalid range bytes", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		//Now safe to send final headers and push the payload out
		respWriter.Header().Add("Content-Range", fmt.Sprintf("%d-%d/%d", startb, endb, size))
		respWriter.WriteHeader(http.StatusPartialContent)

		_, _ = io.CopyN(respWriter, reader, endb-startb+1)
	} else {
		_, _ = io.Copy(respWriter, reader)
	}

}
func (server *Server) httpHandleIndex(respWriter http.ResponseWriter, _ *http.Request) {
	respWriter.Header().Set("Content-Type", "text/html; charset=UTF-8")
	err := server.webui.RenderGameListing(respWriter)

	if err != nil {
		http.Error(respWriter, "Sending file failed", http.StatusInternalServerError)
		return
	}
}
func (server *Server) httpHandleCSS(respWriter http.ResponseWriter, _ *http.Request) {
	respWriter.Header().Set("Content-Type", "text/css")
	_, err := respWriter.Write(webui.SkeletonCss)
	if err != nil {
		http.Error(respWriter, "Sending file failed", http.StatusInternalServerError)
		return
	}
}
func (server *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(res, "Only GET is allowed", http.StatusMethodNotAllowed)
		return
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

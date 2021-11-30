package server

import (
	"fmt"

	"github.com/ralim/switchhost/library"
	"github.com/ralim/switchhost/server/virtualftp"
	titledb "github.com/ralim/switchhost/titledb"
	"github.com/ralim/switchhost/webui"
)

//Server is the main server that renders out the files in the database

type Server struct {
	library  *library.Library
	webui    *webui.WebUI
	httpPort int
	ftpPort  int
}

func NewServer(lib *library.Library, titledb *titledb.TitlesDB, httpPort, ftpPort int) *Server {
	return &Server{httpPort: httpPort, ftpPort: ftpPort, library: lib, webui: webui.NewWebUI(lib, titledb)}
}

func (server *Server) Run() {
	fmt.Println("Starting servers")
	go virtualftp.StartFTP(server.library, server.ftpPort)
	server.StartHTTP()

}

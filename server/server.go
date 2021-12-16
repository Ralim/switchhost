package server

import (
	"fmt"

	"github.com/ralim/switchhost/library"
	"github.com/ralim/switchhost/server/virtualftp"
	"github.com/ralim/switchhost/settings"
	titledb "github.com/ralim/switchhost/titledb"
	"github.com/ralim/switchhost/webui"
)

//Server is the main server that renders out the files in the database

type Server struct {
	library  *library.Library
	webui    *webui.WebUI
	settings *settings.Settings
}

func NewServer(lib *library.Library, titledb *titledb.TitlesDB, settings *settings.Settings) *Server {
	return &Server{
		library:  lib,
		webui:    webui.NewWebUI(lib, titledb),
		settings: settings,
	}
}

func (server *Server) Run() {
	fmt.Println("Starting servers")
	go virtualftp.StartFTP(server.library, server.settings)
	server.StartHTTP()

}

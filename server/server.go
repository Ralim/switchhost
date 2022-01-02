package server

import (
	"context"
	"net/http"

	"github.com/ralim/switchhost/library"
	"github.com/ralim/switchhost/server/virtualftp"
	"github.com/ralim/switchhost/settings"
	"github.com/ralim/switchhost/titledb"
	"github.com/ralim/switchhost/webui"
	"github.com/rs/zerolog/log"
)

//Server is the main server that renders out the files in the database

type Server struct {
	library  *library.Library
	webui    *webui.WebUI
	settings *settings.Settings
	titledb  *titledb.TitlesDB

	httpServer *http.Server
	ftpServer  *virtualftp.FTPServer
}

func NewServer(lib *library.Library, titledb *titledb.TitlesDB, settings *settings.Settings) *Server {
	return &Server{
		library:  lib,
		webui:    webui.NewWebUI(lib, titledb),
		settings: settings,
		titledb:  titledb,
	}
}

func (server *Server) Run() {
	log.Info().Msg("Starting servers, press ctrl-c to exit cleanly")

	server.ftpServer = virtualftp.CreateVirtualFTP(server.library, server.settings)
	go server.StartHTTP()
	go server.ftpServer.Start()

}

func (server *Server) Stop() {
	if server.httpServer != nil {
		server.httpServer.Shutdown(context.Background())
		log.Info().Msg("HTTP task exiting")
	}
	if server.ftpServer != nil {
		server.ftpServer.Stop()
		log.Info().Msg("FTP task exiting")
	}
}

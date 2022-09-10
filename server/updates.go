package server

import (
	"encoding/json"
	"net/http"
)

func (server *Server) handleServingUpdatesList(respWriter http.ResponseWriter, req *http.Request) {

	updates := server.library.GetGamesNeedingUpdate()
	data, _ := json.MarshalIndent(updates, "", "  ")
	_, _ = respWriter.Write(data)

}

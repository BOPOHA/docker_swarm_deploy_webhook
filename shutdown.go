package main

import (
	"net/http"
)

const byebyeMessage = "BYE-BYE!"
const shutdownEnpoint = "/shutdown"

type shutdownHandler struct {
	server *http.Server
}

func (h *shutdownHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	go withouterrIOClose(h.server)
	LogRespWriter(w.Write([]byte(byebyeMessage)))
}

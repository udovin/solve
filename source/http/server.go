package http

import (
	"fmt"
	"net/http"

	"../config"
)

type Server struct {
	server http.Server
}

func NewServer(cfg *config.ServerConfig) *Server {
	server := http.Server{}
	server.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	return &Server{server: server}
}

func (s *Server) Listen() error {
	return s.server.ListenAndServe()
}

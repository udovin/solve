package http

import (
	"fmt"
	"net/http"

	"../app"
	"../config"
)

type Server struct {
	server http.Server
	app    *app.App
}

func NewServer(cfg *config.Config) (*Server, error) {
	a, err := app.NewApp(cfg)
	if err != nil {
		return nil, err
	}
	server := Server{
		server: setupHttpServer(&cfg.Server),
		app:    a,
	}
	return &server, nil
}

func setupHttpServer(cfg *config.ServerConfig) http.Server {
	server := http.Server{}
	server.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	return server
}

func (s *Server) Listen() error {
	s.app.Start()
	defer s.app.Stop()
	return s.server.ListenAndServe()
}

package http

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"../app"
	"../config"
)

type Server struct {
	app    *app.App
	logger *log.Logger
	server http.Server
	router http.ServeMux
}

func NewServer(cfg *config.Config) (*Server, error) {
	solve, err := app.NewApp(cfg)
	if err != nil {
		return nil, err
	}
	server := http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
	}
	s := Server{
		app:    solve,
		logger: log.New(os.Stdout, "[http] ", log.LstdFlags),
		server: server,
	}
	s.server.SetKeepAlivesEnabled(true)
	s.server.Handler = http.HandlerFunc(s.handler)
	return &s, nil
}

func (s *Server) Listen() error {
	s.app.Start()
	defer s.app.Stop()
	return s.server.ListenAndServe()
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		s.logger.Println(r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
	}()
	s.router.ServeHTTP(w, r)
}

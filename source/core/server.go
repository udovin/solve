package core

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"../config"
)

type Server struct {
	logger *log.Logger
	server http.Server
	router http.ServeMux
}

func NewServer(cfg *config.ServerConfig) (*Server, error) {
	server := http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
	}
	s := Server{
		logger: log.New(os.Stdout, "[http] ", log.LstdFlags),
		server: server,
	}
	s.server.SetKeepAlivesEnabled(true)
	s.server.Handler = http.HandlerFunc(s.handler)
	static := http.FileServer(http.Dir("static"))
	s.router.Handle("/static/", http.StripPrefix("/static/", static))
	return &s, nil
}

func (s *Server) Handle(pattern string, handler http.HandlerFunc) {
	s.router.Handle(pattern, handler)
}

func (s *Server) Listen() error {
	return s.server.ListenAndServe()
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		s.logger.Println(r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
	}()
	s.router.ServeHTTP(w, r)
}

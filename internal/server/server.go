package server

import (
	"net/http"
	"time"
)

type Registrar interface {
	Register(mux *http.ServeMux)
}

type Server struct {
	http *http.Server
}

func New(addr string, registrars ...Registrar) *Server {
	mux := http.NewServeMux()
	for _, r := range registrars {
		r.Register(mux)
	}

	return &Server{
		http: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Run() error {
	return s.http.ListenAndServe()
}

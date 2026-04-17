package server

import (
	"net/http"
	"time"
)

// Server wraps http.Server to manage the application's HTTP lifecycle.
type Server struct {
	httpServer *http.Server
}

// Run configures and starts the HTTP server on the given port.
func (s *Server) Run(port string, handler http.Handler) error {
	s.httpServer = &http.Server{
		Addr:           port,
		Handler:        handler,
		MaxHeaderBytes: 1 << 20,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
	}

	return s.httpServer.ListenAndServe()
}

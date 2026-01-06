package http

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Server represents the HTTP server
type Server struct {
	server *http.Server
}

// NewServer creates a new HTTP server
func NewServer(handler http.Handler, port int) *Server {
	return &Server{
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Start starts the HTTP server (non-blocking - returns when server exits)
func (s *Server) Start() error {
	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Addr returns the server address
func (s *Server) Addr() string {
	return s.server.Addr
}

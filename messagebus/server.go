package messagebus

import (
	"pulsyflux/socket"
)

// Server wraps socket server for message bus
type Server struct {
	server *socket.Server
}

// NewServer creates a new message bus server
func NewServer(port string) *Server {
	return &Server{
		server: socket.NewServer(port),
	}
}

// Start starts the server
func (s *Server) Start() error {
	return s.server.Start()
}

// Stop stops the server
func (s *Server) Stop() error {
	return s.server.Stop()
}

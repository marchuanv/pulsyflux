package httpcontainers

import (
	"pulsyflux/contracts"
)

type httpServer struct {
	address        *contracts.URI
	readTimeout    *contracts.TimeDuration
	writeTimeout   *contracts.TimeDuration
	maxHeaderBytes int
	handler        *contracts.HttpRequestHandler
}

func (s *httpServer) GetAddress() *contracts.URI {
	return s.address
}

func (s *httpServer) SetAddress(addr *contracts.URI) {
	s.address = addr
}

func (s *httpServer) GetReadTimeout() *contracts.TimeDuration {
	return s.readTimeout
}

func (s *httpServer) SetReadTimeout(duration *contracts.TimeDuration) {
	s.readTimeout = duration
}

func (s *httpServer) GetWriteTimeout() *contracts.TimeDuration {
	return s.writeTimeout
}

func (s *httpServer) SetWriteTimeout(duration *contracts.TimeDuration) {
	s.writeTimeout = duration
}

func (s *httpServer) GetMaxHeaderBytes() int {
	return s.maxHeaderBytes
}

func (s *httpServer) SetMaxHeaderBytes(size int) {
	s.maxHeaderBytes = size
}

func (s *httpServer) GetHandler() *contracts.HttpRequestHandler {
	return s.handler
}

func (s *httpServer) SetHandler(handler *contracts.HttpRequestHandler) {
	s.handler = handler
}

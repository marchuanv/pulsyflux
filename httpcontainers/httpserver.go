package httpcontainers

import (
	"context"
	"log"
	"net/http"
	"pulsyflux/contracts"
	"time"
)

var server = &http.Server{}

type httpServer struct {
	address        *uri
	readTimeout    *timeDuration
	writeTimeout   *timeDuration
	maxHeaderBytes *int
	handler        *httpRequestHandler
}

func (s *httpServer) GetAddress() contracts.URI {
	return s.address
}

func (s *httpServer) SetAddress(addr contracts.URI) {
	s.address = addr.(*uri)
}

func (s *httpServer) GetReadTimeout() contracts.TimeDuration {
	return s.readTimeout
}

func (s *httpServer) SetReadTimeout(duration contracts.TimeDuration) {
	s.readTimeout = duration.(*timeDuration)
}

func (s *httpServer) GetWriteTimeout() contracts.TimeDuration {
	return s.writeTimeout
}

func (s *httpServer) SetWriteTimeout(duration contracts.TimeDuration) {
	s.writeTimeout = duration.(*timeDuration)
}

func (s *httpServer) GetMaxHeaderBytes() int {
	return *s.maxHeaderBytes
}

func (s *httpServer) SetMaxHeaderBytes(size *int) {
	s.maxHeaderBytes = size
}

func (s *httpServer) GetHandler() contracts.HttpRequestHandler {
	return s.handler
}

func (s *httpServer) SetHandler(handler contracts.HttpRequestHandler) {
	s.handler = handler.(*httpRequestHandler)
}

func (s *httpServer) Start() {
	if server.Addr == "" {
		server = &http.Server{
			Addr:           s.GetAddress().GetHostAddress(),
			ReadTimeout:    s.GetReadTimeout().GetDuration(),
			WriteTimeout:   s.GetWriteTimeout().GetDuration(),
			MaxHeaderBytes: s.GetMaxHeaderBytes(),
			Handler:        s.GetHandler(),
		}
	}
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func (s *httpServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped.")
	server = &http.Server{}
}

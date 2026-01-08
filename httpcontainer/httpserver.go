package httpcontainer

import (
	"context"
	"log"
	"net"
	"net/http"
	"pulsyflux/shared"
	"sync"
	"time"
)

type serverState int32

const (
	stateStopped serverState = iota
	stateStarting
	stateRunning
	stateStopping
)

var (
	singletonMu     sync.Mutex
	singletonServer *httpServer
)

type httpServer struct {
	// immutable config (set once)
	address         *uri
	readTimeout     *timeDuration
	writeTimeout    *timeDuration
	idleTimeout     *timeDuration
	responseTimeout *timeDuration
	maxHeaderBytes  maxHeaderBytes
	httpReqHCon     *httpReqHandler

	// runtime state
	mu       sync.Mutex
	state    serverState
	server   *http.Server
	listener net.Listener
	done     chan struct{}
}

func (s *httpServer) GetAddress() shared.URI {
	return s.address
}

func (s *httpServer) GetResponseHandler(msgId shared.MsgId) shared.HttpResHandler {
	return s.httpReqHCon.getResHandler(msgId)
}

func (s *httpServer) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == stateRunning || s.state == stateStarting {
		return
	}

	if s.state == stateStopping {
		done := s.done
		s.mu.Unlock()
		if done != nil {
			<-done
		}
		s.mu.Lock()
	}

	hostAddr := s.address.GetHostAddress()

	ln, err := net.Listen("tcp", hostAddr)
	if err != nil {
		log.Printf("http server listen failed on %s: %v", hostAddr, err)
		return
	}

	handler := http.TimeoutHandler(
		s.httpReqHCon,
		s.responseTimeout.GetDuration(),
		"server timeout",
	)

	s.server = &http.Server{
		Addr:           hostAddr,
		Handler:        handler,
		ReadTimeout:    s.readTimeout.GetDuration(),
		WriteTimeout:   s.writeTimeout.GetDuration(),
		IdleTimeout:    s.idleTimeout.GetDuration(),
		MaxHeaderBytes: int(s.maxHeaderBytes),
	}

	s.listener = ln
	s.done = make(chan struct{})
	s.state = stateRunning

	go s.serve()
}

func (s *httpServer) serve() {
	err := s.server.Serve(s.listener)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil && err != http.ErrServerClosed {
		log.Printf("http server error: %v", err)
	}

	close(s.done)
	s.server = nil
	s.listener = nil
	s.state = stateStopped
}

func (s *httpServer) Stop() {
	s.mu.Lock()

	if s.state != stateRunning {
		s.mu.Unlock()
		return
	}

	s.state = stateStopping
	server := s.server
	done := s.done

	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("http server shutdown error: %v", err)
	}

	if done != nil {
		<-done
	}
}

func newHttpServer(
	addr shared.URI,
	readTimeout shared.ReadTimeDuration,
	writeTimeout shared.WriteTimeDuration,
	idleTimeout shared.IdleConnTimeoutDuration,
	responseTimeout shared.ResponseTimeoutDuration,
	maxHeaderBytes maxHeaderBytes,
	httpReqHCon shared.HttpReqHandler,
) shared.HttpServer {

	singletonMu.Lock()
	defer singletonMu.Unlock()

	if singletonServer != nil {
		return singletonServer
	}

	singletonServer = &httpServer{
		address:         addr.(*uri),
		readTimeout:     readTimeout.(*timeDuration),
		writeTimeout:    writeTimeout.(*timeDuration),
		idleTimeout:     idleTimeout.(*timeDuration),
		responseTimeout: responseTimeout.(*timeDuration),
		maxHeaderBytes:  maxHeaderBytes,
		httpReqHCon:     httpReqHCon.(*httpReqHandler),
		state:           stateStopped,
	}

	return singletonServer
}

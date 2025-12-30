package httpcontainer

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"pulsyflux/contracts"
	"sync"
	"time"
)

type serverState int

const (
	stateStopped serverState = iota
	stateStarting
	stateRunning
	stateStopping
)

var (
	server            http.Server
	serverOnce        sync.Once
	connMu            sync.Mutex
	listener          net.Listener
	serverStateGlobal serverState
	serveDoneGlobal   chan struct{}
)

type httpServer struct {
	address         *uri
	readTimeout     *timeDuration
	writeTimeout    *timeDuration
	idleTimeout     *timeDuration
	responseTimeout *timeDuration
	maxHeaderBytes  maxHeaderBytes
	httpReqHCon     *httpReqHandler
}

func (s *httpServer) GetAddress() contracts.URI {
	return s.address
}

func (s *httpServer) GetResponseHandler(msgId contracts.MsgId) contracts.HttpResHandler {
	return s.httpReqHCon.getResHandler(msgId)
}

func (s *httpServer) Start() {
	connMu.Lock()
	defer connMu.Unlock()

	// If already running or starting, return
	if serverStateGlobal == stateRunning || serverStateGlobal == stateStarting {
		return
	}

	// If stopping, wait until shutdown completes
	if serverStateGlobal == stateStopping && serveDoneGlobal != nil {
		done := serveDoneGlobal
		connMu.Unlock()
		<-done
		connMu.Lock()
	}

	serverStateGlobal = stateStarting
	hostAddr := s.address.GetHostAddress()

	// Initialize global server only once
	serverOnce.Do(func() {
		server = http.Server{
			Addr:           hostAddr,
			Handler:        http.TimeoutHandler(s.httpReqHCon, s.responseTimeout.GetDuration(), "server timeout"),
			ReadTimeout:    s.readTimeout.GetDuration(),
			WriteTimeout:   s.writeTimeout.GetDuration(),
			IdleTimeout:    s.idleTimeout.GetDuration(),
			MaxHeaderBytes: int(s.maxHeaderBytes),
		}
	})

	// Bind listener
	ln, err := net.Listen("tcp", hostAddr)
	if err != nil {
		serverStateGlobal = stateStopped
		panic(fmt.Sprintf("failed to listen on %s: %v", hostAddr, err))
	}
	listener = ln

	serveDoneGlobal = make(chan struct{})

	// Start Serve in a goroutine
	go func() {
		serverStateGlobal = stateRunning
		err := server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}

		// signal completion
		close(serveDoneGlobal)
		serverStateGlobal = stateStopped
		listener = nil
		serveDoneGlobal = nil
		log.Println("Server gracefully stopped.")
	}()
}

func (s *httpServer) Stop() {
	connMu.Lock()
	defer connMu.Unlock()

	if serverStateGlobal != stateRunning {
		return
	}

	serverStateGlobal = stateStopping
	done := serveDoneGlobal

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = server.Shutdown(ctx)
		if done != nil {
			<-done // wait for Serve() to exit
		}

		log.Println("Server stop completed (async).")
	}()
}

func newHttpServer(
	addr contracts.URI,
	readTimeout contracts.ReadTimeDuration,
	writeTimeout contracts.WriteTimeDuration,
	idleTimeout contracts.IdleConnTimeoutDuration,
	responseTimeout contracts.ResponseTimeoutDuration,
	maxHeaderBytes maxHeaderBytes,
	httpReqHCon contracts.HttpReqHandler,
) contracts.HttpServer {
	return &httpServer{
		address:         addr.(*uri),
		readTimeout:     readTimeout.(*timeDuration),
		writeTimeout:    writeTimeout.(*timeDuration),
		idleTimeout:     idleTimeout.(*timeDuration),
		responseTimeout: responseTimeout.(*timeDuration),
		maxHeaderBytes:  maxHeaderBytes,
		httpReqHCon:     httpReqHCon.(*httpReqHandler),
	}
}

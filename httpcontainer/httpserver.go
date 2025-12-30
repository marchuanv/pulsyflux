package httpcontainer

import (
	"context"
	"log"
	"net/http"
	"pulsyflux/contracts"
	"sync"
	"time"
)

var (
	server     http.Server
	serverOnce sync.Once
)

type httpServer struct {
	address        *uri
	readTimeout    *timeDuration
	writeTimeout   *timeDuration
	idleTimeout    *timeDuration
	maxHeaderBytes maxHeaderBytes
	httpReqHCon    *httpReqHandler
}

func (s *httpServer) GetAddress() contracts.URI {
	return s.address
}

func (s *httpServer) Start() {
	serverOnce.Do(func() {
		server = http.Server{
			Addr:           s.address.GetHostAddress(),
			ReadTimeout:    s.readTimeout.GetDuration(),
			WriteTimeout:   s.writeTimeout.GetDuration(),
			IdleTimeout:    s.idleTimeout.GetDuration(),
			MaxHeaderBytes: int(s.maxHeaderBytes),
			Handler:        s.httpReqHCon,
		}
	})
	err := server.ListenAndServe()
	if err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server closed under request")
		} else {
			panic(err)
		}
	}
}

func (s *httpServer) GetResponseHandler(msgId contracts.MsgId) contracts.HttpResHandler {
	return s.httpReqHCon.getResHandler(msgId)
}

func (s *httpServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := server.Shutdown(ctx)
	if err == nil {
		log.Println("Server gracefully stopped.")
	} else {
		if err == context.DeadlineExceeded {
			log.Printf("Server shutdown timed out: forced exit after 5s")
		} else {
			log.Printf("Server shutdown failed: %v", err)
		}
	}
}

func newHttpServer(
	addr contracts.URI,
	readTimeout contracts.ReadTimeDuration,
	writeTimeout contracts.WriteTimeDuration,
	idleTimeout contracts.IdleConnTimeoutDuration,
	maxHeaderBytes maxHeaderBytes,
	httpReqHCon contracts.HttpReqHandler,
) contracts.HttpServer {
	return &httpServer{
		address:        addr.(*uri),
		readTimeout:    readTimeout.(*timeDuration),
		writeTimeout:   writeTimeout.(*timeDuration),
		idleTimeout:    idleTimeout.(*timeDuration),
		maxHeaderBytes: maxHeaderBytes,
		httpReqHCon:    httpReqHCon.(*httpReqHandler),
	}
}

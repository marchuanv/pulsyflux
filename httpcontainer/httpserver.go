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
			MaxHeaderBytes: int(s.maxHeaderBytes),
			Handler:        s.httpReqHCon,
		}
	})
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func (s *httpServer) Response(msgId contracts.MsgId) contracts.HttpResHandler {
	return s.httpReqHCon.getResHandler(msgId)
}

func (s *httpServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped.")
}

func newHttpServer(
	addr contracts.URI,
	readTimeout contracts.ReadTimeDuration,
	writeTimeout contracts.WriteTimeDuration,
	maxHeaderBytes maxHeaderBytes,
	httpReqHCon contracts.HttpReqHandler,
) contracts.HttpServer {
	return &httpServer{
		address:        addr.(*uri),
		readTimeout:    readTimeout.(*timeDuration),
		writeTimeout:   writeTimeout.(*timeDuration),
		maxHeaderBytes: maxHeaderBytes,
		httpReqHCon:    httpReqHCon.(*httpReqHandler),
	}
}

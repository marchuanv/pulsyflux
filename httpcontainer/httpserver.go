package httpcontainer

import (
	"context"
	"log"
	"net"
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

func (s *httpServer) Start() {
	hostAddr := s.address.GetHostAddress()
	serverOnce.Do(func() {
		server = http.Server{
			Addr:           hostAddr,
			ReadTimeout:    s.readTimeout.GetDuration(),
			WriteTimeout:   s.writeTimeout.GetDuration(),
			IdleTimeout:    s.idleTimeout.GetDuration(),
			MaxHeaderBytes: int(s.maxHeaderBytes),
			Handler:        http.TimeoutHandler(s.httpReqHCon, s.responseTimeout.GetDuration(), "server timeout"),
		}
	})
	conn, err := net.DialTimeout("tcp", hostAddr, 5*time.Second)
	if err != nil {
		go func() {
			err := server.ListenAndServe()
			if err != nil {
				if err == http.ErrServerClosed {
					log.Println("Server closed under request")
				} else {
					panic(err)
				}
			}
		}()
	} else {
		conn.Close()
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

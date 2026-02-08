package socket

import (
	"context"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

type Server struct {
	port           string
	ln             net.Listener
	ctx            context.Context
	cancel         context.CancelFunc
	conns          sync.WaitGroup
	requestHandler *requestHandler
	peers          *peers
}

func NewServer(port string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		port:   port,
		ctx:    ctx,
		cancel: cancel,
		peers:  newPeers(),
	}
}

func (s *Server) Start() error {
	lc := &net.ListenConfig{
		Control: func(network, address string, conn syscall.RawConn) error {
			return conn.Control(func(fd uintptr) {
				// Set SO_REUSEADDR to allow quick rebinding
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				// Increase socket receive buffer to 2MB
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 2*1024*1024)
				// Increase socket send buffer to 2MB
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 2*1024*1024)
			})
		},
	}
	ln, err := lc.Listen(context.Background(), "tcp4", "127.0.0.1:"+s.port)
	if err != nil {
		return err
	}
	s.ln = ln
	s.requestHandler = newRequestHandler(64, 8192, s.peers) // Increased worker pool and queue size for better concurrency
	go s.acceptLoop()
	return nil
}

func (s *Server) Stop() error {
	s.cancel()
	s.ln.Close()
	s.conns.Wait()
	s.requestHandler.stop()
	s.peers = nil
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				continue
			}
		}
		s.conns.Add(1)
		// Use a separate goroutine for each connection handler
		// to ensure accepts don't block on handler processing
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer s.conns.Done()

	ctx := &connctx{
		conn:   conn,
		writes: make(chan *frame, 8192), // Increased from 2048 to handle more concurrent writes
		errors: make(chan *frame, 2048), // Increased from 512 for better error delivery
		closed: make(chan struct{}),
		wg:     &sync.WaitGroup{},
	}
	ctx.wg.Add(1)
	go ctx.startWriter()

	streamReqs := make(map[uuid.UUID]*request)

	var currentClientID uuid.UUID

	defer func() {
		// Cancel all pending requests
		for _, req := range streamReqs {
			if req.cancel != nil {
				req.cancel()
			}
		}
		s.peers.delete(currentClientID)
		close(ctx.writes)
		close(ctx.errors)
		ctx.wg.Wait()
		conn.Close()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		f, err := ctx.readFrame()
		if err != nil {
			return
		}

		currentClientID = f.ClientID
		s.peers.set(currentClientID, ctx) // Register client connection

		switch f.Type {
		case startFrame:
			timeout := time.Duration(f.ClientTimeoutMs) * time.Millisecond
			reqCtx, cancel := context.WithTimeout(context.Background(), timeout)
			req := &request{
				connctx:      ctx,
				frame:        f,
				requestID:    f.RequestID,
				clientID:     currentClientID,
				peerClientID: f.PeerClientID,
				timeout:      timeout,
				ctx:          reqCtx,
				cancel:       cancel,
			}
			streamReqs[f.RequestID] = req
			if !s.requestHandler.handle(req) {
				req.cancel()
				ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "server overloaded"))
			}

		case chunkFrame:
			req := streamReqs[f.RequestID]
			reqCtx, cancel := context.WithTimeout(context.Background(), req.timeout)
			req.cancel = cancel
			req.ctx = reqCtx
			if !s.requestHandler.handle(req) {
				cancel()
				ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "server overloaded"))
			}

		case endFrame:
			req := streamReqs[f.RequestID]
			delete(streamReqs, f.RequestID)
			reqCtx, cancel := context.WithTimeout(context.Background(), req.timeout)
			req.cancel = cancel
			req.ctx = reqCtx
			if !s.requestHandler.handle(req) {
				cancel()
				ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "server overloaded"))
			}

		default:
			ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "invalid frame type"))
		}
	}
}

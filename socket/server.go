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
	port     string
	ln       net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	conns    sync.WaitGroup
	registry *registry
}

func NewServer(port string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		port:     port,
		ctx:      ctx,
		cancel:   cancel,
		registry: newRegistry(),
	}
}

func (s *Server) Start() error {
	lc := &net.ListenConfig{
		Control: func(network, address string, conn syscall.RawConn) error {
			return conn.Control(func(fd uintptr) {
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 2*1024*1024)
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 2*1024*1024)
			})
		},
	}
	ln, err := lc.Listen(context.Background(), "tcp4", "127.0.0.1:"+s.port)
	if err != nil {
		return err
	}
	s.ln = ln
	go s.acceptLoop()
	return nil
}

func (s *Server) Stop() error {
	s.cancel()
	s.ln.Close()
	s.conns.Wait()
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
		writes: make(chan *frame, 1024),
		reads:  make(chan *frame, 1024),
		errors: make(chan *frame, 512),
		closed: make(chan struct{}),
		wg:     &sync.WaitGroup{},
	}
	ctx.wg.Add(2)
	go ctx.startWriter()
	go ctx.startReader()

	var clientID uuid.UUID
	var entry *clientEntry

	defer func() {
		if clientID != uuid.Nil {
			s.registry.unregister(clientID)
		}
		close(ctx.writes)
		close(ctx.errors)
		ctx.wg.Wait()
		conn.Close()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		case f, ok := <-ctx.reads:
			if !ok {
				return
			}

			if clientID == uuid.Nil {
				clientID = f.ClientID
				entry = s.registry.register(clientID, f.ChannelID, ctx)
			}

			if f.Flags&flagReceive != 0 {
				// Client wants to receive a request - dequeue and send directly
				if req, ok := entry.dequeueRequest(); ok {
					entry.connctx.enqueue(req)
				} else {
					// Try peers' queues
					peers := s.registry.getChannelPeers(entry.channelID, clientID)
					found := false
					for _, peer := range peers {
						if req, ok := peer.dequeueRequest(); ok {
							entry.connctx.enqueue(req)
							found = true
							break
						}
					}
					if !found {
						entry.connctx.enqueue(newErrorFrame(uuid.Nil, clientID, "no requests available", 0))
					}
				}
				putFrame(f)
			} else if f.Flags&flagResponse != 0 {
				// Find original requester and send response
				if peer, ok := s.registry.get(f.ClientID); ok {
					peer.enqueueResponse(f)
				} else {
					putFrame(f)
				}
			} else if f.Flags&flagRequest != 0 {
				// Request frame - enqueue to peers' request queues
				peers := s.registry.getChannelPeers(entry.channelID, clientID)
				if len(peers) == 0 {
					if !entry.enqueueRequest(f) {
						entry.enqueueResponse(newErrorFrame(f.RequestID, clientID, "request queue full", 0))
						putFrame(f)
					}
				} else {
					enqueued := false
					for _, peer := range peers {
						if peer.enqueueRequest(f) {
							enqueued = true
						}
					}
					if !enqueued {
						entry.enqueueResponse(newErrorFrame(f.RequestID, clientID, "peer queue full", 0))
					}
					putFrame(f)
				}
			} else {
				// Unknown flag
				putFrame(f)
			}
		}
	}
}

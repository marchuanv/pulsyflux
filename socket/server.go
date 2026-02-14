package socket

import (
	"context"
	"fmt"
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
				fmt.Printf("[Server] Accept error: %v\n", err)
				continue
			}
		}
		fmt.Printf("[Server] Accepted new connection\n")
		s.conns.Add(1)
		// Use a separate goroutine for each connection handler
		// to ensure accepts don't block on handler processing
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer s.conns.Done()
	fmt.Printf("[Server] New connection handler started\n")

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
		fmt.Printf("[Server] Connection handler exiting\n")
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
			fmt.Printf("[Server] Context done\n")
			return
		case f, ok := <-ctx.reads:
			if !ok {
				fmt.Printf("[Server] Reads channel closed\n")
				return
			}
			fmt.Printf("[Server] Received frame type=%d flags=0x%x\n", f.Type, f.Flags)

			if clientID == uuid.Nil {
				clientID = f.ClientID
				entry = s.registry.register(clientID, f.ChannelID, ctx)
			}

			if f.Flags&flagReceive != 0 {
				go func() {
					for {
						select {
						case <-s.ctx.Done():
							putFrame(f)
							return
						default:
						}
						fmt.Printf("[Server] flagReceive type=%d from client=%s\n", f.Type, clientID.String()[:8])
						if req, ok := entry.dequeueRequest(); ok {
							fmt.Printf("[Server] Dequeued from own queue, type=%d\n", req.Type)
							entry.enqueueResponse(req)
							putFrame(f)
							return
						}
						peers := s.registry.getChannelPeers(entry.channelID, clientID)
						fmt.Printf("[Server] Trying %d peers' queues\n", len(peers))
						for _, peer := range peers {
							if req, ok := peer.dequeueRequest(); ok {
								fmt.Printf("[Server] Dequeued from peer queue, type=%d, payloadLen=%d\n", req.Type, len(req.Payload))
								entry.enqueueResponse(req)
								putFrame(f)
								return
							}
						}
						time.Sleep(10 * time.Millisecond)
					}
				}()
			} else if f.Flags&flagResponse != 0 {
				fmt.Printf("[Server] flagResponse type=%d to client=%s\n", f.Type, f.ClientID.String()[:8])
				if peer, ok := s.registry.get(f.ClientID); ok {
					peer.enqueueResponse(f)
				} else {
					putFrame(f)
				}
			} else if f.Flags&flagRequest != 0 {
				fmt.Printf("[Server] flagRequest type=%d from client=%s, payloadLen=%d\n", f.Type, clientID.String()[:8], len(f.Payload))
				peers := s.registry.getChannelPeers(entry.channelID, clientID)
				fmt.Printf("[Server] Found %d peers\n", len(peers))
				if len(peers) == 0 {
					entry.enqueueRequest(f)
				} else {
					for _, peer := range peers {
						peer.enqueueRequest(f)
					}
				}
			} else {
				putFrame(f)
			}
		}
	}
}

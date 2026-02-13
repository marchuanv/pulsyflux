package socket

import (
	"context"
	"net"
	"sync"
	"syscall"

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
				// Client wants to receive - try to dequeue immediately
				if req, ok := entry.dequeueRequest(); ok {
					entry.connctx.enqueue(req)
					putFrame(f)
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
						// No requests available - queue the receive frame
						if !entry.enqueuePendingReceive(f) {
							// Queue full - send error
							entry.connctx.enqueue(newErrorFrame(uuid.Nil, clientID, "pending receive queue full", 0))
							putFrame(f)
						}
						// If enqueue succeeded, don't putFrame - keep it queued
					} else {
						putFrame(f)
					}
				}
			} else if f.Flags&flagResponse != 0 {
				// Response frame - route back to original requester
				if peer, ok := s.registry.get(f.ClientID); ok {
					select {
					case peer.connctx.writes <- f:
					case <-s.ctx.Done():
						putFrame(f)
						return
					}
				} else {
					putFrame(f)
				}
			} else if f.Flags&flagRequest != 0 {
				// Route request to peers
				peers := s.registry.getChannelPeers(entry.channelID, clientID)
				sent := false
				
				// For START frames, try pending receives first
				if f.Type == startFrame {
					for _, peer := range peers {
						if _, ok := peer.dequeuePendingReceive(); ok {
							select {
							case peer.connctx.writes <- f:
								sent = true
							default:
							}
							if sent {
								break
							}
						}
					}
				} else {
					// For CHUNK/END frames, send directly to first peer
					if len(peers) > 0 {
						select {
						case peers[0].connctx.writes <- f:
							sent = true
						default:
						}
					}
				}
				
				if !sent {
					// No pending receives - enqueue to request queues
					if len(peers) == 0 {
						// No peers - enqueue to own queue and send ACK
						if !entry.enqueueRequest(f) {
							entry.enqueueResponse(newErrorFrame(f.RequestID, clientID, "request queue full", 0))
							putFrame(f)
						} else if f.Type == startFrame {
							// Send ACK for start frame
							ack := getFrame()
							ack.Version = version1
							ack.Type = startFrame
							ack.RequestID = f.RequestID
							ack.ClientID = clientID
							ack.ChannelID = entry.channelID
							ack.Flags = flagResponse
							select {
							case entry.connctx.writes <- ack:
							case <-s.ctx.Done():
								putFrame(ack)
								return
							}
						}
					} else if f.Type == startFrame {
						// START frame with peers - enqueue to all peers
						enqueued := false
						for _, peer := range peers {
							if peer.enqueueRequest(f) {
								enqueued = true
							}
						}
						if !enqueued {
							entry.enqueueResponse(newErrorFrame(f.RequestID, clientID, "peer queue full", 0))
						} else {
							// Send ACK since request was enqueued
							ack := getFrame()
							ack.Version = version1
							ack.Type = startFrame
							ack.RequestID = f.RequestID
							ack.ClientID = clientID
							ack.ChannelID = entry.channelID
							ack.Flags = flagResponse
							select {
							case entry.connctx.writes <- ack:
							case <-s.ctx.Done():
								putFrame(ack)
								return
							}
						}
						putFrame(f)
					}
				}
			} else {
				// Unknown flag
				putFrame(f)
			}
		}
	}
}

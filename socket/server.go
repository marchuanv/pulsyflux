package socket

import (
	"context"
	"log"
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
		reads:  make(chan *frame, 100),
		errors: make(chan *frame, 2048), // Increased from 512 for better error delivery
		closed: make(chan struct{}),
		wg:     &sync.WaitGroup{},
	}
	ctx.wg.Add(2)
	go ctx.startWriter()
	go ctx.startReader()

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
		case f, ok := <-ctx.reads:
			if !ok {
				return
			}

			switch f.Type {
			case startFrame:
				log.Printf("[Server] Received START frame for request %s from client %s", f.RequestID, f.ClientID)
				timeout := time.Duration(f.ClientTimeoutMs) * time.Millisecond
				reqCtx, cancel := context.WithTimeout(context.Background(), timeout)
				req := &request{
					connctx:      ctx,
					frame:        f,
					requestID:    f.RequestID,
					clientID:     f.ClientID,
					peerClientID: f.PeerClientID,
					timeout:      timeout,
					ctx:          reqCtx,
					cancel:       cancel,
				}

				currentClientID = f.ClientID
				switch f.Flags {
				case flagHandshakeStarted:
					log.Printf("[Server] Handshake started: registering client %s for channel %s", f.ClientID, f.ChannelID)
					s.peers.set(currentClientID, ctx, f.ChannelID)
					peer := s.peers.pair(currentClientID, f.ChannelID)
					if peer == nil {
						log.Printf("[Server] No peer available for client %s", currentClientID)
						ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "no peer client available", flagPeerNotAvailable))
						req.cancel()
					} else {
						log.Printf("[Server] Found peer %s for client %s", peer.clientID, currentClientID)
						req.peerClientID = peer.clientID
						if !s.requestHandler.handle(req) {
							req.cancel()
							ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "server overloaded", 0))
							return
						}
					}
				case flagHandshakeCompleted:
					log.Printf("[Server] Handshake completed: request ID mapping service ready for client %s", currentClientID)
					req.peerClientID = f.PeerClientID
					currentPeer, _ := s.peers.get(currentClientID)
					peer, _ := s.peers.get(f.PeerClientID)

					close(currentPeer.ready)
					<-peer.ready

					log.Printf("[Server] Both clients %s and %s are ready, processing request %s", currentClientID, f.PeerClientID, f.RequestID)

					if !s.requestHandler.handle(req) {
						req.cancel()
						ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "server overloaded", 0))
						return
					}
				default:
					req.peerClientID = f.PeerClientID
					currentPeer, _ := s.peers.get(currentClientID)
					peer, _ := s.peers.get(f.PeerClientID)
					if pendingReqID, found := peer.mapper.getPending(); found && pendingReqID == f.RequestID {
						currentPeer.mapper.mapRequest(f.RequestID, pendingReqID)
						peer.mapper.mapRequest(pendingReqID, f.RequestID)
						peer.mapper.setPending(uuid.Nil)
						log.Printf("[Server] Mapped response %s -> %s", f.RequestID, pendingReqID)
					} else {
						currentPeer.mapper.setPending(f.RequestID)
						log.Printf("[Server] Stored pending request %s from client %s", f.RequestID, currentClientID)
					}
					streamReqs[f.RequestID] = req
					if !s.requestHandler.handle(req) {
						req.cancel()
						ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "server overloaded", 0))
					}
					return
				}

			case chunkFrame:
				log.Printf("[Server] Received CHUNK frame for request %s from client %s (size=%d)", f.RequestID, f.ClientID, len(f.Payload))
				req := streamReqs[f.RequestID]
				if req == nil {
					log.Printf("[Server] ERROR: Received CHUNK before START for request %s", f.RequestID)
					ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "chunk received before start", 0))
					putFrame(f)
					continue
				}
				req.frame = f
				if !s.requestHandler.handle(req) {
					req.cancel()
					ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "server overloaded", 0))
				}

			case endFrame:
				log.Printf("[Server] Received END frame for request %s from client %s", f.RequestID, f.ClientID)
				req := streamReqs[f.RequestID]
				if req == nil {
					log.Printf("[Server] ERROR: Received END before START for request %s", f.RequestID)
					ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "end received before start", 0))
					putFrame(f)
					continue
				}
				req.frame = f
				delete(streamReqs, f.RequestID)
				// Cancel old context before creating new one
				req.cancel()
				reqCtx, cancel := context.WithTimeout(context.Background(), req.timeout)
				req.cancel = cancel
				req.ctx = reqCtx
				if !s.requestHandler.handle(req) {
					cancel()
					ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "server overloaded", 0))
				}

			default:
				// Route all other frame types directly to peer
				peer, ok := s.peers.get(f.PeerClientID)
				if !ok {
					ctx.enqueue(newErrorFrame(f.RequestID, f.ClientID, "peer not found", flagPeerNotAvailable))
					continue
				}
				peer.connctx.enqueue(f)
			}
		}
	}
}

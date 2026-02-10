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
		reads:  make(chan *frame, 1000), // Increased to prevent blocking
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
				case flagHandshake:
					s.peers.set(currentClientID, ctx, f.ChannelID, f.RequestID)
					peer := s.peers.pair(currentClientID, f.ChannelID)
					if peer == nil {
						req.cancel()
					} else {
						// Send handshake response to current client
						resp := getFrame()
						resp.Version = version1
						resp.Type = startFrame
						resp.RequestID = f.RequestID
						resp.ClientID = peer.clientID
						resp.PeerClientID = currentClientID
						resp.ChannelID = f.ChannelID
						resp.ClientTimeoutMs = f.ClientTimeoutMs
						resp.Flags = flagHandshake
						ctx.enqueue(resp)
						
						// Send handshake notification to peer using peer's request ID
						peerResp := getFrame()
						peerResp.Version = version1
						peerResp.Type = startFrame
						peerResp.RequestID = peer.handshakeReqID
						peerResp.ClientID = currentClientID
						peerResp.PeerClientID = peer.clientID
						peerResp.ChannelID = f.ChannelID
						peerResp.ClientTimeoutMs = f.ClientTimeoutMs
						peerResp.Flags = flagHandshake
						peer.connctx.enqueue(peerResp)
						req.cancel()
					}
				default:
					req.peerClientID = f.PeerClientID
					if f.PeerClientID == uuid.Nil {
						// First request, need to find peer
						currentPeer, ok := s.peers.get(currentClientID)
						if !ok {
							ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "client not registered", 0))
							req.cancel()
							continue
						}
						peer := s.peers.pair(currentClientID, currentPeer.channelID)
						if peer == nil {
							ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "no peer available", flagPeerNotAvailable))
							req.cancel()
							continue
						}
						req.peerClientID = peer.clientID
						f.PeerClientID = peer.clientID
						currentPeer.mapper.setPending(f.RequestID)
					}
					streamReqs[f.RequestID] = req
					if !s.requestHandler.handle(req) {
						req.cancel()
						ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "server overloaded", 0))
					}
				}

			case chunkFrame:
				req := streamReqs[f.RequestID]
				if req == nil {
					ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "chunk received before start", 0))
					putFrame(f)
					continue
				}
				req.frame = f
				// Cancel old context and create new one
				req.cancel()
				reqCtx, cancel := context.WithTimeout(context.Background(), req.timeout)
				req.cancel = cancel
				req.ctx = reqCtx
				if !s.requestHandler.handle(req) {
					req.cancel()
					ctx.enqueue(newErrorFrame(f.RequestID, currentClientID, "server overloaded", 0))
				}

			case endFrame:
				req := streamReqs[f.RequestID]
				if req == nil {
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

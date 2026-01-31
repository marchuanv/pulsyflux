package socket

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

type server struct {
	port           string
	ln             net.Listener
	ctx            context.Context
	cancel         context.CancelFunc
	conns          sync.WaitGroup
	requestHandler *requestHandler
	clientRegistry *clientRegistry
}

func NewServer(port string) *server {
	ctx, cancel := context.WithCancel(context.Background())
	return &server{
		port:   port,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *server) Start() error {
	lc := &net.ListenConfig{
		Control: func(network, address string, conn syscall.RawConn) error {
			return conn.Control(func(fd uintptr) {
				// Set SO_REUSEADDR to allow quick rebinding
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				// Increase socket receive buffer
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 512*1024)
				// Increase socket send buffer
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 512*1024)
			})
		},
	}
	ln, err := lc.Listen(context.Background(), "tcp4", "127.0.0.1:"+s.port)
	if err != nil {
		return err
	}
	s.ln = ln
	registry := newClientRegistry()
	s.clientRegistry = registry
	s.requestHandler = newRequestHandler(64, 8192, registry)
	go s.acceptLoop()
	return nil
}

func (s *server) Stop() error {
	s.cancel()
	s.ln.Close()
	s.conns.Wait()
	s.requestHandler.stop()
	return nil
}

func (s *server) acceptLoop() {
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

func (s *server) handle(conn net.Conn) {
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

	defer func() {
		// Cancel all pending requests
		for reqID, req := range streamReqs {
			streamReqs[reqID] = nil
			s.clientRegistry.removeClient(req.role, req.channelID, req.clientID)
			if req.cancel != nil {
				req.cancel()
			}
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
		default:
		}

		f, err := newFrame(conn)
		if err != nil {
			return
		}

		switch f.Type {
		case StartFrame:
			// Validate payload length
			if len(f.Payload) < 41 {
				ctx.send(newErrorFrame(f.RequestID, "start frame payload too short"))
				continue
			}

			// Extract fields from payload
			role := ClientRole(f.Payload[0])
			if role != RoleConsumer && role != RoleProvider {
				ctx.send(newErrorFrame(f.RequestID, "invalid client role"))
				continue
			}

			timeoutMs := binary.BigEndian.Uint64(f.Payload[1:9])
			timeout := time.Duration(timeoutMs) * time.Millisecond

			var clientID uuid.UUID
			copy(clientID[:], f.Payload[9:25])

			var channelID uuid.UUID
			copy(channelID[:], f.Payload[25:41])

			if !s.clientRegistry.hasClient(role, channelID, clientID) {
				s.clientRegistry.addClient(role, channelID, clientID, ctx)
			}

			// Wait for peer with timeout
			if !s.clientRegistry.hasPeerForChannel(role, channelID) {
				waitCtx, waitCancel := context.WithTimeout(context.Background(), timeout)
				defer waitCancel()

				ticker := time.NewTicker(50 * time.Millisecond)
				defer ticker.Stop()

				for {
					select {
					case <-waitCtx.Done():
						ctx.send(newErrorFrame(f.RequestID, "no peer available for channel"))
						continue
					case <-ticker.C:
						if s.clientRegistry.hasPeerForChannel(role, channelID) {
							goto peerFound
						}
					}
				}
			}

		peerFound:

			// Create the request record without storing peer info
			reqCtx, cancel := context.WithCancel(context.Background())
			streamReqs[f.RequestID] = &request{
				connctx:   ctx,
				frame:     f,
				requestID: f.RequestID,
				clientID:  clientID,
				channelID: channelID,
				payload:   []byte{},
				timeout:   timeout,
				role:      role,
				ctx:       reqCtx,
				cancel:    cancel,
			}
		case ChunkFrame:
			req := streamReqs[f.RequestID]
			req.payload = append(req.payload, f.Payload...)

		case EndFrame:
			req := streamReqs[f.RequestID]
			delete(streamReqs, f.RequestID)

			// Process as new request
			reqCtx, cancel := context.WithTimeout(context.Background(), req.timeout)
			req.ctx = reqCtx
			req.cancel = cancel

			if !s.requestHandler.handle(req) {
				cancel()
				ctx.send(newErrorFrame(req.frame.RequestID, "server overloaded"))
			}

		case ResponseFrame:
			// Extract consumer's clientID and channelID from response payload
			if len(f.Payload) < 32 {
				continue
			}

			var clientID uuid.UUID
			copy(clientID[:], f.Payload[0:16])

			var channelID uuid.UUID
			copy(channelID[:], f.Payload[16:32])

			// Get the consumer's connection
			consumerCtx, ok := s.clientRegistry.getClient(RoleConsumer, channelID, clientID)
			if !ok {
				continue
			}

			// Send response back to consumer
			consumerCtx.send(f)

		case ErrorFrame:
			// Check if error payload contains routing info
			if len(f.Payload) >= 32 {
				var clientID uuid.UUID
				copy(clientID[:], f.Payload[0:16])

				var channelID uuid.UUID
				copy(channelID[:], f.Payload[16:32])

				consumerCtx, ok := s.clientRegistry.getClient(RoleConsumer, channelID, clientID)
				if ok {
					consumerCtx.send(f)
					continue
				}
			}
			// If no routing info or consumer not found, error is for current connection
			ctx.send(f)

		default:
			ctx.send(newErrorFrame(f.RequestID, "invalid message type"))
		}
	}
}

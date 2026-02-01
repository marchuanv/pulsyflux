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

type Server struct {
	port           string
	ln             net.Listener
	ctx            context.Context
	cancel         context.CancelFunc
	conns          sync.WaitGroup
	requestHandler *requestHandler
	clientRegistry *clientRegistry
}

func NewServer(port string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		port:   port,
		ctx:    ctx,
		cancel: cancel,
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
	registry := newClientRegistry()
	s.clientRegistry = registry
	s.requestHandler = newRequestHandler(64, 8192, registry)
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
		errors: make(chan *frame, 2048), // Increased from 512 for better error delivery
		closed: make(chan struct{}),
		wg:     &sync.WaitGroup{},
	}
	ctx.wg.Add(1)
	go ctx.startWriter()

	streamReqs := make(map[uuid.UUID]*request)
	responseRouting := make(map[uuid.UUID]*connctx)

	var registeredRole clientRole
	var registeredClientID uuid.UUID
	var registeredChannelID uuid.UUID
	var clientRegistered bool

	defer func() {
		// Cancel all pending requests
		for _, req := range streamReqs {
			if req.cancel != nil {
				req.cancel()
			}
		}
		// Remove client from registry if registered
		if clientRegistered {
			s.clientRegistry.removeClient(registeredRole, registeredChannelID, registeredClientID)
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
		case startFrame:
			// Validate payload length
			if len(f.Payload) != 41 {
				ctx.send(newErrorFrame(f.RequestID, "invalid start frame payload length"))
				continue
			}

			// Extract fields from payload
			role := clientRole(f.Payload[0])
			if role != roleConsumer && role != roleProvider {
				ctx.send(newErrorFrame(f.RequestID, "invalid client role"))
				continue
			}

			timeoutMs := binary.BigEndian.Uint64(f.Payload[1:9])
			timeout := time.Duration(timeoutMs) * time.Millisecond

			var clientID uuid.UUID
			copy(clientID[:], f.Payload[9:25])

			var channelID uuid.UUID
			copy(channelID[:], f.Payload[25:41])

			isRegistration := f.Flags&flagRegistration != 0
			reqCtx, cancel := context.WithTimeout(context.Background(), timeout)
			req := &request{
				connctx:        ctx,
				frame:          f,
				requestID:      f.RequestID,
				clientID:       clientID,
				channelID:      channelID,
				timeout:        timeout,
				role:           role,
				ctx:            reqCtx,
				cancel:         cancel,
				isRegistration: isRegistration,
			}
			streamReqs[f.RequestID] = req

			isClientRegistered := s.clientRegistry.hasClient(role, channelID, clientID)

			if !isClientRegistered {
				s.clientRegistry.addClient(role, channelID, clientID, ctx)
				clientRegistered = true
				registeredRole = role
				registeredClientID = clientID
				registeredChannelID = channelID
			}

			if !isRegistration {
				if !s.requestHandler.handle(req) {
					req.cancel()
					ctx.send(newErrorFrame(f.RequestID, "server overloaded"))
				}
			}

		case chunkFrame:
			req := streamReqs[f.RequestID]
			if req != nil && !req.isRegistration {
				reqCopy := *req
				reqCopy.frame = f
				if !s.requestHandler.handle(&reqCopy) {
					reqCopy.cancel()
					ctx.send(newErrorFrame(f.RequestID, "server overloaded"))
				}
			}

		case endFrame:
			req := streamReqs[f.RequestID]
			delete(streamReqs, f.RequestID)

			if req != nil && !req.isRegistration {
				reqCopy := *req
				reqCopy.frame = f
				if !s.requestHandler.handle(&reqCopy) {
					reqCopy.cancel()
					ctx.send(newErrorFrame(f.RequestID, "server overloaded"))
				}
			}

		case responseStartFrame:
			if len(f.Payload) < 32 {
				continue
			}

			var clientID uuid.UUID
			copy(clientID[:], f.Payload[0:16])

			var channelID uuid.UUID
			copy(channelID[:], f.Payload[16:32])

			consumerCtx, ok := s.clientRegistry.getClient(roleConsumer, channelID, clientID)
			if !ok {
				continue
			}

			responseRouting[f.RequestID] = consumerCtx
			consumerCtx.send(f)

		case responseChunkFrame:
			consumerCtx, ok := responseRouting[f.RequestID]
			if !ok {
				continue
			}
			consumerCtx.send(f)

		case responseEndFrame:
			consumerCtx, ok := responseRouting[f.RequestID]
			delete(responseRouting, f.RequestID)
			if !ok {
				continue
			}
			consumerCtx.send(f)

		case responseFrame:
			// Extract consumer's clientID and channelID from response payload
			if len(f.Payload) < 32 {
				continue
			}

			var clientID uuid.UUID
			copy(clientID[:], f.Payload[0:16])

			var channelID uuid.UUID
			copy(channelID[:], f.Payload[16:32])

			// Get the consumer's connection
			consumerCtx, ok := s.clientRegistry.getClient(roleConsumer, channelID, clientID)
			if !ok {
				continue
			}

			// Send response back to consumer
			consumerCtx.send(f)

		case errorFrame:
			// Check if error payload contains routing info
			if len(f.Payload) >= 32 {
				var clientID uuid.UUID
				copy(clientID[:], f.Payload[0:16])

				var channelID uuid.UUID
				copy(channelID[:], f.Payload[16:32])

				consumerCtx, ok := s.clientRegistry.getClient(roleConsumer, channelID, clientID)
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

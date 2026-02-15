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
	var channelID uuid.UUID
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
				channelID = f.ChannelID
				entry = s.registry.register(clientID, channelID, ctx)
				// Send registration ack
				ackF := getFrame()
				ackF.Version = version1
				ackF.Type = ackFrame
				ackF.Flags = flagNone
				ackF.ClientID = clientID
				entry.enqueueResponse(ackF)
			}

			if f.Flags&flagAck != 0 {
				s.registry.recordAck(f.FrameID)
				putFrame(f)
			} else if f.Flags&flagRequest != 0 {
				s.handleRequest(f, clientID, channelID, entry)
			} else {
				putFrame(f)
			}
		}
	}
}

func (s *Server) handleRequest(f *frame, clientID, channelID uuid.UUID, entry *clientEntry) {
	peers := s.registry.waitForReceivers(channelID, clientID, f.ClientTimeoutMs)
	if len(peers) == 0 {
		errFrame := newErrorFrame(f.RequestID, f.ClientID, "no receivers available", flagNone)
		entry.enqueueResponse(errFrame)
		putFrame(f)
		return
	}
	
	frameID := uuid.New()
	f.FrameID = frameID
	collector := s.registry.createAckCollector(frameID, len(peers))
	
	for _, peer := range peers {
		peer.enqueueRequest(cloneFrame(f))
	}
	
	timeout := time.After(time.Duration(f.ClientTimeoutMs) * time.Millisecond)
	select {
	case <-collector.done:
		ackF := getFrame()
		ackF.Version = version1
		ackF.Type = ackFrame
		ackF.Flags = flagAck
		ackF.RequestID = f.RequestID
		ackF.ClientID = f.ClientID
		ackF.FrameID = frameID
		entry.enqueueResponse(ackF)
	case <-timeout:
		errFrame := newErrorFrame(f.RequestID, f.ClientID, "timeout waiting for acknowledgments", flagNone)
		entry.enqueueResponse(errFrame)
	}
	
	s.registry.removeAckCollector(frameID)
	putFrame(f)
}

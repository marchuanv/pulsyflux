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
				log.Printf("[Server] Registering client %s on channel %s", clientID.String()[:8], channelID.String()[:8])
				entry = s.registry.register(clientID, channelID, ctx)
			}

			if f.Flags&flagAck != 0 {
				log.Printf("[Server] Received ack from client %s, frameID=%s", clientID.String()[:8], f.FrameID.String()[:8])
				s.registry.recordAck(f.FrameID)
				putFrame(f)
			} else if f.Flags&flagReceive != 0 {
				log.Printf("[Server] Received registration/ready frame from client %s", clientID.String()[:8])
				putFrame(f)
			} else if f.Flags&flagResponse != 0 {
				log.Printf("[Server] Received response from client %s", clientID.String()[:8])
				if peer, ok := s.registry.get(f.ClientID); ok {
					peer.enqueueResponse(f)
				} else {
					putFrame(f)
				}
			} else if f.Flags&flagRequest != 0 {
				log.Printf("[Server] Received request from client %s, reqID=%s", clientID.String()[:8], f.RequestID.String()[:8])
				
				timeout := time.After(time.Duration(f.ClientTimeoutMs) * time.Millisecond)
				ticker := time.NewTicker(100 * time.Millisecond)
				
				var peers []*clientEntry
				timedOut := false
				for {
					peers = s.registry.getChannelPeers(channelID, clientID)
					if len(peers) > 0 {
						break
					}
					
					select {
					case <-timeout:
						timedOut = true
						break
					case <-ticker.C:
					}
					if timedOut {
						break
					}
				}
				ticker.Stop()
				
				if timedOut {
					errFrame := newErrorFrame(f.RequestID, f.ClientID, "no receivers available", flagResponse)
					entry.enqueueResponse(errFrame)
					putFrame(f)
				} else {
					frameID := uuid.New()
					f.FrameID = frameID
					collector := s.registry.createAckCollector(frameID, f.RequestID, clientID, f.Type, len(peers))
					
					for _, peer := range peers {
						clonedFrame := cloneFrame(f)
						peer.enqueueRequest(clonedFrame)
					}
					
					<-collector.done
					
					ackF := getFrame()
					ackF.Version = version1
					ackF.Type = ackFrame
					ackF.Flags = flagAck
					ackF.RequestID = f.RequestID
					ackF.ClientID = f.ClientID
					ackF.FrameID = frameID
					entry.enqueueResponse(ackF)
					s.registry.removeAckCollector(frameID)
					putFrame(f)
				}
			} else {
				putFrame(f)
			}
		}
	}
}

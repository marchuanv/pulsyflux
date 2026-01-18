package socket

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"time"
)

type server struct {
	port   string
	ln     net.Listener
	ctx    context.Context
	cancel context.CancelFunc
	conns  sync.WaitGroup
	pool   *workerpool
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
	ln, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return err
	}
	s.ln = ln
	s.pool = newWorkerPool(16, 2048)
	go s.acceptLoop()
	return nil
}

func (s *server) Stop(ctx context.Context) error {
	s.cancel()
	s.ln.Close()
	s.conns.Wait()
	s.pool.stop()
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
		go s.handle(conn)
	}
}

func (s *server) handle(conn net.Conn) {
	defer s.conns.Done()

	ctx := &connctx{
		conn:   conn,
		writes: make(chan *frame, 2048),
		closed: make(chan struct{}),
		wg:     &sync.WaitGroup{},
	}
	ctx.wg.Add(1)
	startWriter(ctx)
	defer func() {
		close(ctx.writes)
		ctx.wg.Wait()
		conn.Close()
	}()

	streamReqs := make(map[uint64]*request)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		f, err := readFrame(conn)
		if err != nil {
			return
		}

		switch f.Type {
		case StartFrame:
			if len(f.Payload) != 8 {
				ctx.send(errorFrame(f.RequestID, "invalid request start header"))
				continue
			}
			timeoutMs := binary.BigEndian.Uint64(f.Payload)
			streamReqs[f.RequestID] = &request{
				connctx:   ctx,
				frame:     f,
				payload:   []byte{},
				timeout:   time.Duration(timeoutMs) * time.Millisecond,
				requestID: f.RequestID,
			}

		case ChunkFrame:
			req := streamReqs[f.RequestID]
			req.payload = append(req.payload, f.Payload...)

		case EndFrame:
			req := streamReqs[f.RequestID]
			delete(streamReqs, f.RequestID)

			reqCtx, cancel := context.WithTimeout(context.Background(), req.timeout)
			req.ctx = reqCtx
			req.cancel = cancel

			if !s.pool.submit(*req) {
				cancel()
				ctx.send(errorFrame(req.frame.RequestID, "server overloaded"))
			}

		default:
			ctx.send(errorFrame(f.RequestID, "invalid message type"))
		}
	}
}

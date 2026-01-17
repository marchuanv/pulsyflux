package socket

import (
	"context"
	"encoding/json"
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
	s.pool = newWorkerPool(8, 1024)

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
		writes: make(chan *frame, 64),
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

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(readTimeout))
		frame, err := readFrame(conn)
		if err != nil {
			return
		}

		if frame.Type != MsgRequest {
			ctx.send(errorFrame(frame.RequestID, "invalid message type"))
			continue
		}

		// Parse client-specified timeout
		timeout := defaultReqTimeout
		var payload requestpayload
		if err := json.Unmarshal(frame.Payload, &payload); err == nil && payload.TimeoutMs > 0 {
			clientTimeout := time.Duration(payload.TimeoutMs) * time.Millisecond
			if clientTimeout < defaultReqTimeout {
				timeout = clientTimeout
			}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), timeout)

		req := request{
			ctx,
			frame,
			reqCtx,
			cancel,
		}

		if !s.pool.submit(req) {
			cancel()
			ctx.send(errorFrame(frame.RequestID, "server overloaded"))
		}
	}
}

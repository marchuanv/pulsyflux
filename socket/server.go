package socket

import (
	"bytes"
	"context"
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

	// Maps for streaming requests
	streamBuffers := make(map[uint64][]byte)
	streamMetas := make(map[uint64]requestmeta)

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

		switch frame.Type {

		case MsgRequestStart:
			// Start streaming request
			meta, err := decodeRequestMeta(bytes.NewReader(frame.Payload))
			if err != nil {
				ctx.send(errorFrame(frame.RequestID, "invalid meta"))
				continue
			}

			streamBuffers[frame.RequestID] = make([]byte, 0, meta.DataSize)
			streamMetas[frame.RequestID] = *meta

		case MsgRequestChunk:
			buf := streamBuffers[frame.RequestID]
			buf = append(buf, frame.Payload...)
			streamBuffers[frame.RequestID] = buf

		case MsgRequestEnd:
			payload := streamBuffers[frame.RequestID]
			meta := streamMetas[frame.RequestID]

			timeout := defaultReqTimeout
			if meta.TimeoutMs > 0 {
				clientTimeout := time.Duration(meta.TimeoutMs) * time.Millisecond
				if clientTimeout < defaultReqTimeout {
					timeout = clientTimeout
				}
			}

			reqCtx, cancel := context.WithTimeout(context.Background(), timeout)
			req := request{
				connctx: ctx,
				frame:   frame,
				meta:    meta,
				payload: payload,
				ctx:     reqCtx,
				cancel:  cancel,
			}

			// Clean up streaming maps
			delete(streamBuffers, frame.RequestID)
			delete(streamMetas, frame.RequestID)

			if !s.pool.submit(req) {
				cancel()
				ctx.send(errorFrame(frame.RequestID, "server overloaded"))
			}

		default:
			ctx.send(errorFrame(frame.RequestID, "invalid message type"))
		}
	}
}

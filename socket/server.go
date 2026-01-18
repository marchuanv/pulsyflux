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

// ---------------- Handle a single client connection ----------------
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

	// Map for streaming processors
	streamProcessors := make(map[uint64]*streamprocessor)

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

		// ---------------- Inline small requests ----------------
		case MsgRequest:
			var payload requestpayload
			if err := json.Unmarshal(frame.Payload, &payload); err != nil {
				ctx.send(errorFrame(frame.RequestID, "invalid payload"))
				continue
			}

			timeout := defaultReqTimeout
			if payload.TimeoutMs > 0 {
				clientTimeout := time.Duration(payload.TimeoutMs) * time.Millisecond
				if clientTimeout < defaultReqTimeout {
					timeout = clientTimeout
				}
			}

			reqCtx, cancel := context.WithTimeout(context.Background(), timeout)
			req := request{
				connctx: ctx,
				frame:   frame,
				payload: []byte(payload.Data),
				ctx:     reqCtx,
				cancel:  cancel,
			}

			if !s.pool.submit(req) {
				cancel()
				ctx.send(errorFrame(frame.RequestID, "server overloaded"))
			}

		// ---------------- Streaming start ----------------
		case MsgRequestStart:
			var meta requestmeta
			if err := json.Unmarshal(frame.Payload, &meta); err != nil {
				ctx.send(errorFrame(frame.RequestID, "invalid meta"))
				continue
			}
			processor := &streamprocessor{
				RequestID: frame.RequestID,
			}
			streamProcessors[frame.RequestID] = processor

		// ---------------- Streaming chunk ----------------
		case MsgRequestChunk:
			processor, ok := streamProcessors[frame.RequestID]
			if !ok {
				ctx.send(errorFrame(frame.RequestID, "unknown stream ID"))
				continue
			}
			if err := processor.ProcessChunk(frame.Payload); err != nil {
				ctx.send(errorFrame(frame.RequestID, "failed to process chunk"))
				delete(streamProcessors, frame.RequestID)
			}

		// ---------------- Streaming end ----------------
		case MsgRequestEnd:
			processor, ok := streamProcessors[frame.RequestID]
			if !ok {
				ctx.send(errorFrame(frame.RequestID, "unknown stream ID"))
				continue
			}
			resp, err := processor.Finish()
			if err != nil {
				ctx.send(errorFrame(frame.RequestID, err.Error()))
			} else {
				ctx.send(resp)
			}
			delete(streamProcessors, frame.RequestID)

		// ---------------- Invalid message type ----------------
		default:
			ctx.send(errorFrame(frame.RequestID, "invalid message type"))
		}
	}
}

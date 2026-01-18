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

// NewServer creates a new TCP server
func NewServer(port string) *server {
	ctx, cancel := context.WithCancel(context.Background())
	return &server{
		port:   port,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins listening and processing clients
func (s *server) Start() error {
	ln, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return err
	}
	s.ln = ln
	s.pool = newWorkerPool(8, 1024) // optional worker pool
	go s.acceptLoop()
	return nil
}

// Stop shuts down the server
func (s *server) Stop(ctx context.Context) error {
	s.cancel()
	s.ln.Close()
	s.conns.Wait()
	s.pool.stop()
	return nil
}

// acceptLoop handles incoming connections
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

// handle a single client connection
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

	// Maps to store stream processors and their contexts per request
	streamProcessors := make(map[uint64]*streamprocessor)
	streamContexts := make(map[uint64]context.Context)
	streamCancels := make(map[uint64]context.CancelFunc)

	cleanupStream := func(reqID uint64) {
		if c, ok := streamCancels[reqID]; ok {
			c() // cancel context
			delete(streamCancels, reqID)
		}
		delete(streamProcessors, reqID)
		delete(streamContexts, reqID)
	}

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
			// Start streaming request and parse optional meta
			meta := requestmeta{}
			_ = json.Unmarshal(frame.Payload, &meta)

			// Apply client timeout if specified
			timeout := defaultReqTimeout
			if meta.TimeoutMs > 0 {
				clientTimeout := time.Duration(meta.TimeoutMs) * time.Millisecond
				if clientTimeout < defaultReqTimeout {
					timeout = clientTimeout
				}
			}

			// Create a single request-scoped context
			reqCtx, cancel := context.WithTimeout(context.Background(), timeout)

			sp := &streamprocessor{RequestID: frame.RequestID}
			streamProcessors[frame.RequestID] = sp
			streamContexts[frame.RequestID] = reqCtx
			streamCancels[frame.RequestID] = cancel

		case MsgRequestChunk:
			sp, ok := streamProcessors[frame.RequestID]
			if !ok {
				ctx.send(errorFrame(frame.RequestID, "unknown request ID"))
				continue
			}
			reqCtx := streamContexts[frame.RequestID] // use the original request context

			if err := sp.ProcessChunk(reqCtx, frame.Payload); err != nil {
				// Context done (timeout) or error during chunk processing
				resp, _ := sp.Finish(reqCtx)
				ctx.send(resp)
				cleanupStream(frame.RequestID)
				continue
			}

		case MsgRequestEnd:
			sp, ok := streamProcessors[frame.RequestID]
			if !ok {
				ctx.send(errorFrame(frame.RequestID, "unknown request ID"))
				continue
			}
			reqCtx := streamContexts[frame.RequestID]

			resp, _ := sp.Finish(reqCtx) // returns MsgError if timeout
			ctx.send(resp)
			cleanupStream(frame.RequestID)

		default:
			ctx.send(errorFrame(frame.RequestID, "invalid message type"))
		}
	}
}

package socket

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"
)

// client represents a streaming client
type client struct {
	conn      net.Conn
	requestID uint64
}

var ErrRequestIDMismatch = errors.New("response ID mismatch")

// NewClient creates a new streaming client connected to the given port
func NewClient(port string) (*client, error) {
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		return nil, err
	}
	return &client{conn: conn}, nil
}

// Close closes the client connection
func (c *client) Close() error {
	return c.conn.Close()
}

// SendStreamFromReader streams a payload from an io.Reader to the server
func (c *client) SendStreamFromReader(r io.Reader, reqTimeout time.Duration) (*frame, error) {
	reqID := atomic.AddUint64(&c.requestID, 1)

	// --- StartFrame: send timeout as header (8 bytes) ---
	timeoutBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeoutBytes, uint64(reqTimeout.Milliseconds()))

	startFrame := frame{
		Version:   Version1,
		Type:      StartFrame,
		Flags:     0,
		RequestID: reqID,
		Payload:   timeoutBytes,
	}

	if err := writeFrame(c.conn, &startFrame); err != nil {
		return nil, err
	}

	// --- Stream chunks ---
	chunkBuf := make([]byte, maxFrameSize)
	for {
		n, err := r.Read(chunkBuf)
		if n > 0 {
			chunk := frame{
				Version:   Version1,
				Type:      ChunkFrame,
				Flags:     0,
				RequestID: reqID,
				Payload:   chunkBuf[:n],
			}
			if err := writeFrame(c.conn, &chunk); err != nil {
				return nil, err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	// --- EndFrame ---
	endFrame := frame{
		Version:   Version1,
		Type:      EndFrame,
		Flags:     0,
		RequestID: reqID,
	}

	if err := writeFrame(c.conn, &endFrame); err != nil {
		return nil, err
	}

	// --- Wait for server response (using client-side context) ---
	ctx, cancel := context.WithTimeout(context.Background(), reqTimeout+time.Second)
	defer cancel()

	respCh := make(chan *frame, 1)
	errCh := make(chan error, 1)

	go func() {
		resp, err := readFrame(c.conn)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case resp := <-respCh:
		if resp.RequestID != reqID {
			return nil, ErrRequestIDMismatch
		}
		return resp, nil
	}
}

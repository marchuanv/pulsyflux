package socket

import (
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
func (c *client) SendStreamFromReader(r io.Reader, dataSize uint64, timeoutMs uint32) (*frame, error) {
	reqID := atomic.AddUint64(&c.requestID, 1)

	// Build requestmeta
	meta := requestmeta{
		TimeoutMs: timeoutMs,
		DataSize:  dataSize,
		Type:      "json",
		Encoding:  "json",
		Streaming: true,
	}

	// Encode meta using package-private helper
	metaPayload, err := encodeRequestMeta(meta)
	if err != nil {
		return nil, err
	}

	// 1️⃣ Send MsgRequestStart
	startFrame := frame{
		Version:   Version1,
		Type:      MsgRequestStart,
		Flags:     0,
		RequestID: reqID,
		Payload:   metaPayload,
	}
	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := writeFrame(c.conn, &startFrame); err != nil {
		return nil, err
	}

	// 2️⃣ Send chunks
	chunkBuf := make([]byte, maxFrameSize)
	for {
		n, err := r.Read(chunkBuf)
		if n > 0 {
			chunk := frame{
				Version:   Version1,
				Type:      MsgRequestChunk,
				Flags:     0,
				RequestID: reqID,
				Payload:   chunkBuf[:n],
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
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

	// 3️⃣ Send MsgRequestEnd
	endFrame := frame{
		Version:   Version1,
		Type:      MsgRequestEnd,
		Flags:     0,
		RequestID: reqID,
		Payload:   nil,
	}
	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := writeFrame(c.conn, &endFrame); err != nil {
		return nil, err
	}

	// 4️⃣ Wait for server response
	c.conn.SetReadDeadline(time.Now().Add(time.Duration(timeoutMs)*time.Millisecond + 1*time.Second))
	resp, err := readFrame(c.conn)
	if err != nil {
		return nil, err
	}

	// Validate request ID
	if resp.RequestID != reqID {
		return nil, ErrRequestIDMismatch
	}

	return resp, nil
}

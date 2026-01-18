package socket

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

type client struct {
	conn      net.Conn
	requestID uint64
}

func NewClient(port string) (*client, error) {
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		return nil, err
	}
	return &client{conn: conn}, nil
}

func (c *client) Close() error {
	return c.conn.Close()
}

// SendStreamFromReader sends data from any io.Reader in frames (auto-chunked)
func (c *client) SendStreamFromReader(reader io.Reader, timeoutMs uint32) (*frame, error) {
	reqID := atomic.AddUint64(&c.requestID, 1)

	// Build and send stream metadata (Start frame)
	meta := requestmeta{
		TimeoutMs: timeoutMs,
		Streaming: true,
		Type:      "json", // optional, adjust as needed
		Encoding:  "json",
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	startFrame := frame{
		Version:   Version1,
		Type:      MsgRequestStart,
		RequestID: reqID,
		Payload:   metaBytes,
	}

	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := writeFrame(c.conn, &startFrame); err != nil {
		return nil, err
	}

	// Read from reader and send chunk frames
	buf := make([]byte, maxFrameSize)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			chunkFrame := frame{
				Version:   Version1,
				Type:      MsgRequestChunk,
				RequestID: reqID,
				Payload:   buf[:n],
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := writeFrame(c.conn, &chunkFrame); err != nil {
				return nil, err
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	// Send End frame
	endFrame := frame{
		Version:   Version1,
		Type:      MsgRequestEnd,
		RequestID: reqID,
	}
	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := writeFrame(c.conn, &endFrame); err != nil {
		return nil, err
	}

	// Wait for response
	c.conn.SetReadDeadline(time.Now().Add(time.Duration(timeoutMs)*time.Millisecond + 1*time.Second))
	resp, err := readFrame(c.conn)
	if err != nil {
		return nil, err
	}

	if resp.RequestID != reqID {
		return nil, fmt.Errorf("response ID mismatch: got %d, expected %d", resp.RequestID, reqID)
	}
	return resp, nil
}

// Helper: convenience wrapper for sending a small string as a stream
func (c *client) SendString(data string, timeoutMs uint32) (*frame, error) {
	return c.SendStreamFromReader(strings.NewReader(data), timeoutMs)
}

package socket

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"
)

// ---------------- Client ----------------
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

// ---------------- Small inline request ----------------
func (c *client) SendRequest(data string, timeoutMs uint32) (*frame, error) {
	if len(data) > maxFrameSize {
		return nil, fmt.Errorf("payload too large; use SendStreamFromReader")
	}

	reqID := atomic.AddUint64(&c.requestID, 1)

	payloadObj := requestpayload{
		TimeoutMs: timeoutMs,
		Data:      data,
	}
	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return nil, err
	}

	frame := frame{
		Version:   Version1,
		Type:      MsgRequest,
		RequestID: reqID,
		Payload:   payloadBytes,
	}

	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := writeFrame(c.conn, &frame); err != nil {
		return nil, err
	}

	c.conn.SetReadDeadline(time.Now().Add(time.Duration(timeoutMs)*time.Millisecond + 50*time.Millisecond))
	resp, err := readFrame(c.conn)
	if err != nil {
		return nil, err
	}

	if resp.RequestID != reqID {
		return nil, fmt.Errorf("response ID mismatch: got %d, expected %d", resp.RequestID, reqID)
	}
	return resp, nil
}

func (c *client) SendStreamFromReader(reader io.Reader, timeoutMs uint32) (*frame, error) {
	reqID := atomic.AddUint64(&c.requestID, 1)

	// Send stream metadata
	meta := requestmeta{
		TimeoutMs: timeoutMs,
		Streaming: true,
	}
	metaPayload, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	startFrame := frame{
		Version:   Version1,
		Type:      MsgRequestStart,
		RequestID: reqID,
		Payload:   metaPayload,
	}
	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := writeFrame(c.conn, &startFrame); err != nil {
		return nil, err
	}

	// Automatically read from reader in chunks
	buf := make([]byte, maxFrameSize)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			chunkFrame := frame{
				Version:   Version1,
				Type:      MsgRequestChunk,
				RequestID: reqID,
				Payload:   buf[:n], // only send bytes read
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

	// Send end-of-stream frame
	endFrame := frame{
		Version:   Version1,
		Type:      MsgRequestEnd,
		RequestID: reqID,
	}
	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := writeFrame(c.conn, &endFrame); err != nil {
		return nil, err
	}

	// Wait for server response
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

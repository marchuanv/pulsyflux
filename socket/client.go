package socket

import (
	"encoding/json"
	"fmt"
	"net"
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

// SendRequest sends a request; automatically streams if payload > maxFrameSize
func (c *client) SendRequest(data string, timeoutMs uint32) (*frame, error) {
	reqID := atomic.AddUint64(&c.requestID, 1)

	fmt.Printf("[Client] Sending request RequestID=%d, size=%d bytes\n", reqID, len(data))

	rawBytes := []byte(data)

	// Decide whether to stream or inline
	if len(rawBytes) <= maxFrameSize {
		// Inline frame (small payload)
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
			Flags:     0,
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

	// ---- Streaming for large payloads ----
	meta := requestmeta{
		TimeoutMs: timeoutMs,
		DataSize:  uint64(len(rawBytes)), // only the raw payload length
		Type:      "json",
		Encoding:  "json",
	}

	// Build meta payload (no data included here; chunks will follow)
	metaPayload, err := meta.payload(nil)
	if err != nil {
		return nil, err
	}

	// 1. Send MsgRequestStart
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

	// 2. Send chunks
	chunkSize := maxFrameSize
	for offset := 0; offset < len(rawBytes); offset += chunkSize {
		end := offset + chunkSize
		if end > len(rawBytes) {
			end = len(rawBytes)
		}
		chunk := frame{
			Version:   Version1,
			Type:      MsgRequestChunk,
			Flags:     0,
			RequestID: reqID,
			Payload:   rawBytes[offset:end],
		}
		c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := writeFrame(c.conn, &chunk); err != nil {
			return nil, err
		}
	}

	// 3. Send MsgRequestEnd
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

	// 4. Wait for server response
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

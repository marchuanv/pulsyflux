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

// Send a request with a client timeout (ms)
func (c *client) SendRequest(data string, timeoutMs uint32) (*frame, error) {

	// Generate unique RequestID
	reqID := atomic.AddUint64(&c.requestID, 1)

	fmt.Printf("[Client] Sending request with RequestID=%d, data=%q, timeout=%dms\n", reqID, data, timeoutMs)

	// Prepare payload
	payloadObj := requestpayload{
		TimeoutMs: timeoutMs,
		Data:      data,
	}
	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return nil, err
	}

	// Build frame
	frame := frame{
		Version:   Version1,
		Type:      MsgRequest,
		Flags:     0,
		RequestID: reqID,
		Payload:   payloadBytes,
	}

	// Write frame
	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := writeFrame(c.conn, &frame); err != nil {
		return nil, err
	}

	// Read response
	c.conn.SetReadDeadline(time.Now().Add(time.Duration(timeoutMs)*time.Millisecond + 50*time.Millisecond))

	resp, err := readFrame(c.conn)
	if err != nil {
		return nil, err
	}

	// Match request ID
	if resp.RequestID != reqID {
		return nil, fmt.Errorf("response ID mismatch: got %d, expected %d", resp.RequestID, reqID)
	}

	return resp, nil
}

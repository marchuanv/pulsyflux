package socket

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRequestIDMismatch = errors.New("response ID mismatch")
	defaultTimeout       = 5 * time.Second
)

// client represents a logical client for a single role + channel
type client struct {
	addr      string
	conn      net.Conn
	connMu    sync.Mutex
	clientID  uuid.UUID
	role      ClientRole
	channelID uuid.UUID
}

// NewClient automatically creates a TCP connection internally
func NewClient(addr string, role ClientRole, channelID uuid.UUID) (*client, error) {
	c := &client{
		addr:      addr,
		clientID:  uuid.New(),
		role:      role,
		channelID: channelID,
	}
	if err := c.ensureConnection(); err != nil {
		return nil, err
	}
	return c, nil
}

// ensureConnection creates the TCP connection if it does not exist
func (c *client) ensureConnection() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		return nil // already connected
	}

	conn, err := net.Dial("tcp4", c.addr)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *client) Close() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// SendStreamFromReader uses the internal connection, role, and channelID automatically
func (c *client) SendStreamFromReader(r io.Reader, reqTimeout time.Duration) (*frame, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, err
	}

	c.connMu.Lock()
	defer c.connMu.Unlock()

	reqID := uuid.New()
	timeoutMs := uint64(reqTimeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	// Payload: Role (1) + Timeout (8) + ClientID (16) + ChannelID (16)
	payload := make([]byte, 1+8+16+16)
	payload[0] = byte(c.role)
	binary.BigEndian.PutUint64(payload[1:9], timeoutMs)
	copy(payload[9:25], c.clientID[:])
	copy(payload[25:41], c.channelID[:])

	// Build and send StartFrame
	startFrame := frame{
		Version:   Version1,
		Type:      StartFrame,
		Flags:     0,
		RequestID: reqID,
		Payload:   payload,
	}
	if err := startFrame.writeFrame(c.conn); err != nil {
		return nil, err
	}

	// Stream chunks
	buf := make([]byte, maxFrameSize)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := frame{
				Version:   Version1,
				Type:      ChunkFrame,
				Flags:     0,
				RequestID: reqID,
				Payload:   buf[:n],
			}
			if err := chunk.writeFrame(c.conn); err != nil {
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

	// Send EndFrame
	endFrame := frame{
		Version:   Version1,
		Type:      EndFrame,
		Flags:     0,
		RequestID: reqID,
	}
	if err := endFrame.writeFrame(c.conn); err != nil {
		return nil, err
	}

	// Wait for response
	ctx, cancel := context.WithTimeout(context.Background(), reqTimeout+time.Second)
	defer cancel()

	respCh := make(chan *frame, 1)
	errCh := make(chan error, 1)

	go func() {
		resp, err := newFrame(c.conn)
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
		if resp.Type == ErrorFrame {
			return nil, errors.New(string(resp.Payload))
		}
		return resp, nil
	}
}

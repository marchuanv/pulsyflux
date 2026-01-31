package socket

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
)

type Consumer struct {
	baseClient
}

func NewConsumer(addr string, channelID uuid.UUID) (*Consumer, error) {
	c := &Consumer{
		baseClient: baseClient{
			addr:      addr,
			clientID:  uuid.New(),
			channelID: channelID,
			role:      RoleConsumer,
		},
	}
	if err := c.dial(); err != nil {
		return nil, err
	}
	if err := c.register(); err != nil {
		c.close()
		return nil, err
	}
	return c, nil
}

func (c *Consumer) Close() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.close()
}

func (c *Consumer) Send(r io.Reader, reqTimeout time.Duration) ([]byte, error) {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	reqID := uuid.New()
	timeoutMs := uint64(reqTimeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	startFrame := frame{
		Version:   Version1,
		Type:      StartFrame,
		Flags:     0, // Request flag (not registration)
		RequestID: reqID,
		Payload:   c.buildMetadataPayload(timeoutMs),
	}
	if err := startFrame.write(c.conn); err != nil {
		return nil, err
	}

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
			if err := chunk.write(c.conn); err != nil {
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

	endFrame := frame{
		Version:   Version1,
		Type:      EndFrame,
		Flags:     0,
		RequestID: reqID,
	}
	if err := endFrame.write(c.conn); err != nil {
		return nil, err
	}

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
			// Skip routing info if present
			errorMsg := resp.Payload
			if len(errorMsg) >= 32 {
				errorMsg = errorMsg[32:]
			}
			return nil, errors.New(string(errorMsg))
		}
		// Skip routing info from response payload
		if len(resp.Payload) >= 32 {
			return resp.Payload[32:], nil
		}
		return resp.Payload, nil
	}
}

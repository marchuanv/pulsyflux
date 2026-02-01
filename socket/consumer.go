package socket

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

type consumer struct {
	baseClient
}

func NewConsumer(addr string, channelID uuid.UUID) (*consumer, error) {
	c := &consumer{
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

func (c *consumer) Close() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.close()
}

func (c *consumer) Send(r io.Reader, reqTimeout time.Duration) (io.Reader, error) {
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

	if err := c.sendChunkedRequest(reqID, r); err != nil {
		return nil, err
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

	respCh := make(chan io.Reader, 1)
	errCh := make(chan error, 1)

	go func() {
		reader, err := c.receiveChunkedResponse(reqID)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- reader
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case payload := <-respCh:
		return payload, nil
	}
}

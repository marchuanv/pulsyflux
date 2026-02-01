package socket

import (
	"context"
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
			role:      roleConsumer,
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

func (c *Consumer) Send(r io.Reader, reqTimeout time.Duration) (io.Reader, error) {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	reqID := uuid.New()
	timeoutMs := uint64(reqTimeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	_startFrame := getFrame()
	_startFrame.Version = version1
	_startFrame.Type = startFrame
	_startFrame.Flags = 0
	_startFrame.RequestID = reqID
	_startFrame.Payload = c.buildMetadataPayload(timeoutMs)
	if err := _startFrame.write(c.conn); err != nil {
		putFrame(_startFrame)
		return nil, err
	}
	putFrame(_startFrame)

	if err := c.sendChunkedRequest(reqID, r); err != nil {
		return nil, err
	}

	_endFrame := getFrame()
	_endFrame.Version = version1
	_endFrame.Type = endFrame
	_endFrame.Flags = 0
	_endFrame.RequestID = reqID
	_endFrame.Payload = nil
	if err := _endFrame.write(c.conn); err != nil {
		putFrame(_endFrame)
		return nil, err
	}
	putFrame(_endFrame)

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

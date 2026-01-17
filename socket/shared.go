package socket

import (
	"context"
	"encoding/json"
	"time"
)

type requestpayload struct {
	TimeoutMs uint32 `json:"timeout_ms,omitempty"`
	Data      string `json:"data"`
}

type request struct {
	connctx *connctx
	frame   *frame
	ctx     context.Context
	cancel  context.CancelFunc
}

const (
	Version1 byte = 1

	MsgRequest  byte = 0x01
	MsgResponse byte = 0x02
	MsgError    byte = 0x03

	headerSize        = 16
	maxFrameSize      = 1024 * 1024 // 1 MB
	defaultReqTimeout = 500 * time.Millisecond
	readTimeout       = 2 * time.Minute
	writeTimeout      = 5 * time.Second
	queueTimeout      = 100 * time.Millisecond
)

func process(ctx context.Context, req *frame) (*frame, error) {
	// Parse payload
	var payload requestpayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return nil, err
	}

	// Simulate work
	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &frame{
		Version:   Version1,
		Type:      MsgResponse,
		RequestID: req.RequestID,
		Payload:   []byte("ACK: " + payload.Data),
	}, nil
}

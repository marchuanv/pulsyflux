package socket

import (
	"context"
	"encoding/json"
	"fmt"
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
	var pl requestpayload
	if err := json.Unmarshal(req.Payload, &pl); err != nil {
		return nil, err
	}

	// Sleep simulation for testing
	if len(pl.Data) >= 5 && pl.Data[:5] == "sleep" {
		var ms int
		fmt.Sscanf(pl.Data, "sleep %d", &ms)
		select {
		case <-time.After(time.Duration(ms) * time.Millisecond):
			return &frame{
				Version:   Version1,
				Type:      MsgResponse,
				RequestID: req.RequestID,
				Payload:   []byte("Slept successfully"),
			}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Normal processing
	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &frame{
		Version:   Version1,
		Type:      MsgResponse,
		RequestID: req.RequestID,
		Payload:   []byte("ACK: " + pl.Data),
	}, nil

}

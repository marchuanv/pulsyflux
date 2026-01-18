package socket

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// request represents a single server request, small or streamed
type request struct {
	connctx *connctx
	frame   *frame
	meta    requestmeta
	ctx     context.Context
	cancel  context.CancelFunc
	payload []byte // raw payload (small or large)
}

// Legacy small request payload (for backward compatibility)
type requestpayload struct {
	TimeoutMs uint32 `json:"timeout_ms,omitempty"`
	Data      string `json:"data"`
}

// Constants used across the socket package
const (
	Version1 byte = 1

	MsgRequest      byte = 0x01
	MsgResponse     byte = 0x02
	MsgError        byte = 0x03
	MsgRequestStart byte = 0x04 // streaming start
	MsgRequestChunk byte = 0x05 // streaming chunk
	MsgRequestEnd   byte = 0x06 // streaming end

	headerSize        = 16
	maxFrameSize      = 1024 * 1024 // 1 MB for inline frames
	defaultReqTimeout = 500 * time.Millisecond
	readTimeout       = 2 * time.Minute
	writeTimeout      = 5 * time.Second
	queueTimeout      = 100 * time.Millisecond
)

// process handles a request and returns a response frame
func process(ctx context.Context, req request) (*frame, error) {
	data := string(req.payload)

	// Special "sleep" command for testing client timeouts
	if strings.HasPrefix(data, "sleep") {
		var ms int
		fmt.Sscanf(data, "sleep %d", &ms)

		select {
		case <-time.After(time.Duration(ms) * time.Millisecond):
			return &frame{
				Version:   Version1,
				Type:      MsgResponse,
				RequestID: req.frame.RequestID,
				Payload:   []byte("Slept successfully"),
			}, nil
		case <-ctx.Done():
			return errorFrame(req.frame.RequestID, ctx.Err().Error()), ctx.Err()
		}
	}

	// Simulate processing delay
	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		return errorFrame(req.frame.RequestID, ctx.Err().Error()), ctx.Err()
	}

	// Normal ACK response
	return &frame{
		Version:   Version1,
		Type:      MsgResponse,
		RequestID: req.frame.RequestID,
		Payload:   append([]byte("ACK: "), req.payload...),
	}, nil
}

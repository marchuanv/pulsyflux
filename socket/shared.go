package socket

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// -------------------- Request --------------------
type request struct {
	connctx *connctx
	frame   *frame
	meta    requestmeta
	ctx     context.Context
	cancel  context.CancelFunc
	payload []byte // optional, can be nil for streaming
}

// process helper for test commands or small payloads
func process(ctx context.Context, req request) (*frame, error) {
	if req.payload != nil {
		data := string(req.payload)
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
	}

	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		return errorFrame(req.frame.RequestID, ctx.Err().Error()), ctx.Err()
	}

	return &frame{
		Version:   Version1,
		Type:      MsgResponse,
		RequestID: req.frame.RequestID,
		Payload:   append([]byte("ACK: "), req.payload...),
	}, nil
}

// -------------------- Protocol Constants --------------------
const (
	Version1 byte = 1

	MsgResponse     byte = 0x02
	MsgError        byte = 0x03
	MsgRequestStart byte = 0x04 // streaming start
	MsgRequestChunk byte = 0x05 // streaming chunk
	MsgRequestEnd   byte = 0x06 // streaming end

	headerSize        = 16
	maxFrameSize      = 1024 * 1024 // 1 MB per frame
	defaultReqTimeout = 500 * time.Millisecond
	readTimeout       = 2 * time.Minute
	writeTimeout      = 5 * time.Second
	queueTimeout      = 100 * time.Millisecond
)

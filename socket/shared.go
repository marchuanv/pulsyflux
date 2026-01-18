package socket

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// --------------------- Constants ---------------------

// Protocol version
const Version1 byte = 1

// Message types
const (
	MsgRequest      byte = 0x01
	MsgResponse     byte = 0x02
	MsgError        byte = 0x03
	MsgRequestStart byte = 0x04 // streaming start
	MsgRequestChunk byte = 0x05 // streaming chunk
	MsgRequestEnd   byte = 0x06 // streaming end
)

// Frame & payload limits
const (
	headerSize        = 16          // frame header size in bytes
	maxFrameSize      = 1024 * 1024 // 1 MB chunk size
	defaultReqTimeout = 500 * time.Millisecond
	readTimeout       = 2 * time.Minute
	writeTimeout      = 5 * time.Second
	queueTimeout      = 100 * time.Millisecond
)

// --------------------- Types ---------------------

// request represents a single server request (streaming)
type request struct {
	connctx *connctx
	frame   *frame
	meta    requestmeta
	ctx     context.Context
	cancel  context.CancelFunc
	payload []byte
}

// Legacy small request payload (for backward compatibility, unused in streaming)
type requestpayload struct {
	TimeoutMs uint32 `json:"timeout_ms,omitempty"`
	Data      string `json:"data"`
}

// --------------------- Business Logic ---------------------

func process(ctx context.Context, req request) (*frame, error) {
	// Handle sleep commands for testing timeouts
	if strings.HasPrefix(string(req.payload), "sleep") {
		var ms int
		fmt.Sscanf(string(req.payload), "sleep %d", &ms)

		select {
		case <-time.After(time.Duration(ms) * time.Millisecond):
			return &frame{
				Version:   Version1,
				Type:      MsgResponse,
				RequestID: req.frame.RequestID,
				Payload:   []byte("Processed " + strconv.Itoa(len(req.payload)) + " bytes"),
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

	// Normal response for all payloads
	return &frame{
		Version:   Version1,
		Type:      MsgResponse,
		RequestID: req.frame.RequestID,
		Payload:   []byte("Processed " + strconv.Itoa(len(req.payload)) + " bytes"),
	}, nil
}

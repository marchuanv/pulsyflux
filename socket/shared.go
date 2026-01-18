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
const workerQueueTimeout = 100 * time.Millisecond

// Frame & payload limits
const (
	ResponseFrame            byte = 0x02
	ErrorFrame               byte = 0x03
	StartFrame               byte = 0x04        // streaming start
	ChunkFrame               byte = 0x05        // streaming chunk
	EndFrame                 byte = 0x06        // streaming end
	frameHeaderSize               = 16          // frame header size in bytes
	maxFrameSize                  = 1024 * 1024 // 1 MB chunk size
	defaultFrameReadTimeout       = 2 * time.Minute
	defaultFrameWriteTimeout      = 5 * time.Second
)

// --------------------- Types ---------------------

type request struct {
	connctx   *connctx
	frame     *frame        // start frame reference
	requestID uint64        // unique per request
	payload   []byte        // accumulated chunks
	dataSize  uint64        // optional, for preallocation/logging
	timeout   time.Duration // timeout for request
	reqType   string        // e.g., "json"
	encoding  string        // e.g., "json"
	ctx       context.Context
	cancel    context.CancelFunc
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
				Type:      ResponseFrame,
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
		Type:      ResponseFrame,
		RequestID: req.frame.RequestID,
		Payload:   []byte("Processed " + strconv.Itoa(len(req.payload)) + " bytes"),
	}, nil
}

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

var workerQueueTimeout = 100 * time.Millisecond

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
	payloadStr := string(req.payload)

	// Check for sleep command to simulate server-side delay
	if strings.HasPrefix(payloadStr, "sleep") {
		var ms int
		_, err := fmt.Sscanf(payloadStr, "sleep %d", &ms)
		if err != nil {
			return errorFrame(req.frame.RequestID, "invalid sleep command"), err
		}

		select {
		case <-time.After(time.Duration(ms) * time.Millisecond):
			// Return successful response after sleep
			return &frame{
				Version:   Version1,
				Type:      ResponseFrame,
				RequestID: req.frame.RequestID,
				Payload:   []byte("Processed " + strconv.Itoa(len(req.payload)) + " bytes"),
			}, nil
		case <-ctx.Done():
			// Timeout or cancellation
			return errorFrame(req.frame.RequestID, ctx.Err().Error()), ctx.Err()
		}
	}

	// Normal processing: simulate slight delay
	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		return errorFrame(req.frame.RequestID, ctx.Err().Error()), ctx.Err()
	}

	// Return processed response
	return &frame{
		Version:   Version1,
		Type:      ResponseFrame,
		RequestID: req.frame.RequestID,
		Payload:   []byte("Processed " + strconv.Itoa(len(req.payload)) + " bytes"),
	}, nil
}

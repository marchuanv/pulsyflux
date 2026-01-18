package socket

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const Version1 byte = 1

// Worker queue timeout removed from blocking select, handled in submit default
const workerQueueTimeout = 500 * time.Millisecond

const (
	ResponseFrame            byte = 0x02
	ErrorFrame               byte = 0x03
	StartFrame               byte = 0x04
	ChunkFrame               byte = 0x05
	EndFrame                 byte = 0x06
	frameHeaderSize               = 16
	maxFrameSize                  = 1024 * 1024
	defaultFrameReadTimeout       = 2 * time.Minute
	defaultFrameWriteTimeout      = 5 * time.Second
)

type request struct {
	connctx    *connctx
	frame      *frame
	requestID  uint64
	payload    []byte
	dataSize   uint64
	timeout    time.Duration
	reqType    string
	encoding   string
	ctx        context.Context
	cancel     context.CancelFunc
	consumerID uint64
	providerID uint64
}

// process simulates request processing with optional sleep for testing timeouts
func process(ctx context.Context, req request) (*frame, error) {
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

	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		return errorFrame(req.frame.RequestID, ctx.Err().Error()), ctx.Err()
	}

	return &frame{
		Version:   Version1,
		Type:      ResponseFrame,
		RequestID: req.frame.RequestID,
		Payload:   []byte("Processed " + strconv.Itoa(len(req.payload)) + " bytes"),
	}, nil
}

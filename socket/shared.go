package socket

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Worker queue timeout removed from blocking select, handled in submit default
const workerQueueTimeout = 500 * time.Millisecond

type request struct {
	connctx   *connctx      // connection context of the client
	frame     *frame        // Start frame
	requestID uuid.UUID     // per-stream UUID
	clientID  uuid.UUID     // client UUID
	channelID uuid.UUID     // channel UUID (for consumers and providers)
	payload   []byte        // accumulated payload
	dataSize  uint64        // total data size received
	timeout   time.Duration // request timeout
	encoding  string        // optional payload encoding
	ctx       context.Context
	cancel    context.CancelFunc
	role      ClientRole // consumer or provider
}

// // process simulates request processing with optional sleep for testing timeouts
// func process(ctx context.Context, req request) (*frame, error) {
// 	if strings.HasPrefix(string(req.payload), "sleep") {
// 		var ms int
// 		fmt.Sscanf(string(req.payload), "sleep %d", &ms)

// 		select {
// 		case <-time.After(time.Duration(ms) * time.Millisecond):
// 			return &frame{
// 				Version:   Version1,
// 				Type:      ResponseFrame,
// 				RequestID: req.frame.RequestID,
// 				Payload:   []byte("Processed " + strconv.Itoa(len(req.payload)) + " bytes"),
// 			}, nil
// 		case <-ctx.Done():
// 			return errorFrame(req.frame.RequestID, ctx.Err().Error()), ctx.Err()
// 		}
// 	}

// 	select {
// 	case <-time.After(200 * time.Millisecond):
// 	case <-ctx.Done():
// 		return errorFrame(req.frame.RequestID, ctx.Err().Error()), ctx.Err()
// 	}

// 	return &frame{
// 		Version:   Version1,
// 		Type:      ResponseFrame,
// 		RequestID: req.frame.RequestID,
// 		Payload:   []byte("Processed " + strconv.Itoa(len(req.payload)) + " bytes"),
// 	}, nil
// }

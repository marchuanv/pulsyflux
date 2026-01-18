package socket

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type streamprocessor struct {
	RequestID uint64
	Total     int
	// Optional: parsed sleep milliseconds for testing timeouts
	SleepMs int
}

// ProcessChunk handles a chunk and respects the request context
func (p *streamprocessor) ProcessChunk(ctx context.Context, chunk []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		p.Total += len(chunk)

		// Check for "sleep N" command in the first chunk only
		if p.SleepMs == 0 {
			payloadStr := string(chunk)
			if strings.HasPrefix(payloadStr, "sleep ") {
				var ms int
				fmt.Sscanf(payloadStr, "sleep %d", &ms)
				p.SleepMs = ms
			}
		}
	}
	return nil
}

// Finish returns the final response frame, enforcing the context
func (p *streamprocessor) Finish(ctx context.Context) (*frame, error) {
	// If SleepMs is set, simulate the delay
	if p.SleepMs > 0 {
		select {
		case <-time.After(time.Duration(p.SleepMs) * time.Millisecond):
		case <-ctx.Done():
			return &frame{
				Version:   Version1,
				Type:      MsgError,
				RequestID: p.RequestID,
				Payload:   []byte(ctx.Err().Error()),
			}, ctx.Err()
		}
	}

	select {
	case <-ctx.Done():
		return &frame{
			Version:   Version1,
			Type:      MsgError,
			RequestID: p.RequestID,
			Payload:   []byte(ctx.Err().Error()),
		}, ctx.Err()
	default:
		msg := fmt.Sprintf("Processed %d bytes", p.Total)
		return &frame{
			Version:   Version1,
			Type:      MsgResponse,
			RequestID: p.RequestID,
			Payload:   []byte(msg),
		}, nil
	}
}

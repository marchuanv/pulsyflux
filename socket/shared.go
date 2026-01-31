package socket

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRequestIDMismatch = errors.New("response ID mismatch")
	defaultTimeout       = 5 * time.Second
)

const workerQueueTimeout = 500 * time.Millisecond

type request struct {
	connctx   *connctx
	frame     *frame
	requestID uuid.UUID
	clientID  uuid.UUID
	channelID uuid.UUID
	payload   []byte
	dataSize  uint64
	timeout   time.Duration
	encoding  string
	ctx       context.Context
	cancel    context.CancelFunc
	role      ClientRole
}

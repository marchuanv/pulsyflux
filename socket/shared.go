package socket

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	errRequestIDMismatch = errors.New("response ID mismatch")
	defaultTimeout       = 5 * time.Second
	errTimeout            = errors.New("timeout")
	errPeerError          = errors.New("peer error")
	errClosed             = errors.New("client closed")
	errHandshakeFailed    = errors.New("handshake failed")
)

const workerQueueTimeout = 500 * time.Millisecond

type request struct {
	connctx      *connctx
	frame        *frame
	requestID    uuid.UUID
	clientID     uuid.UUID
	peerClientID uuid.UUID
	timeout      time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
}

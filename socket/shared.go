package socket

import (
	"errors"
	"time"
)

var (
	defaultTimeout = 5 * time.Second
	errTimeout     = errors.New("timeout")
	errPeerError   = errors.New("peer error")
	errClosed      = errors.New("client closed")
)

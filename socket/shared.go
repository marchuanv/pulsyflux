package socket

import (
	"errors"
	"time"
)

var (
	defaultTimeout  = 5 * time.Second
	errTimeout      = errors.New("timeout")
	errFrame        = errors.New("error frame received")
	errClosed       = errors.New("client closed")
	errInvalidFrame = errors.New("invalid frame")
)

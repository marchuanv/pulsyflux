package tcpconn

import "errors"

var (
	errConnectionClosed = errors.New("connection closed")
	errConnectionDead   = errors.New("connection dead")
)

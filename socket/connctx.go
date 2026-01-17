package socket

import (
	"net"
	"sync"
	"time"
)

type connctx struct {
	conn   net.Conn
	writes chan *frame
	closed chan struct{}
	wg     *sync.WaitGroup
}

func (ctx *connctx) send(frame *frame) bool {
	select {
	case ctx.writes <- frame:
		return true
	case <-ctx.closed:
		return false
	}
}

func startWriter(ctx *connctx) {
	go func() {
		defer ctx.wg.Done()
		for {
			select {
			case frame, ok := <-ctx.writes:
				if !ok {
					return
				}
				ctx.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
				if err := writeFrame(ctx.conn, frame); err != nil {
					return
				}
			case <-ctx.closed:
				return
			}
		}
	}()
}

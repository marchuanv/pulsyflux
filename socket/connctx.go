package socket

import (
	"net"
	"sync"
)

type connctx struct {
	conn   net.Conn
	writes chan *frame
	closed chan struct{}
	wg     *sync.WaitGroup
}

// send enqueues a frame non-blocking
func (ctx *connctx) send(f *frame) bool {
	select {
	case ctx.writes <- f:
		return true
	case <-ctx.closed:
		return false
	default:
		return false // drop frame if write buffer full
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
				_ = writeFrame(ctx.conn, frame)
			case <-ctx.closed:
				return
			}
		}
	}()
}

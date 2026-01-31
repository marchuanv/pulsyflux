package socket

import (
	"net"
	"sync"
)

type connctx struct {
	conn   net.Conn
	writes chan *frame
	errors chan *frame // Higher priority for error frames
	closed chan struct{}
	wg     *sync.WaitGroup
}

// send enqueues a frame
func (ctx *connctx) send(f *frame) bool {
	defer func() {
		// Recover from panic if channel is closed
		recover()
	}()

	// Send error frames on priority channel
	if f.Type == ErrorFrame {
		select {
		case ctx.errors <- f:
			return true
		case <-ctx.closed:
			return false
		default:
			return false
		}
	}

	// Send regular frames on normal channel
	select {
	case ctx.writes <- f:
		return true
	case <-ctx.closed:
		return false
	default:
		return false // drop frame if write buffer full
	}
}

func (ctx *connctx) startWriter() {
	defer ctx.wg.Done()
	for {
		select {
		// Prioritize error frames
		case frame, ok := <-ctx.errors:
			if !ok {
				// errors channel closed, drain writes and exit
				for {
					select {
					case frame, ok := <-ctx.writes:
						if !ok {
							return
						}
						_ = frame.writeFrame(ctx.conn)
					case <-ctx.closed:
						return
					}
				}
			}
			if frame != nil {
				_ = frame.writeFrame(ctx.conn)
			}
		case frame, ok := <-ctx.writes:
			if !ok {
				return
			}
			_ = frame.writeFrame(ctx.conn)
		case <-ctx.closed:
			return
		}
	}
}

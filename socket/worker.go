package socket

import (
	"sync"
)

type workerpool struct {
	jobs chan request
	wg   sync.WaitGroup
}

func newWorkerPool(workers, queue int) *workerpool {
	wp := &workerpool{
		jobs: make(chan request, queue),
	}

	for i := 0; i < workers; i++ {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			for req := range wp.jobs {
				select {
				case <-req.ctx.Done():
					req.cancel()
					continue
				default:
				}

				resp, err := process(req.ctx, req)
				req.cancel()

				if err != nil {
					req.connctx.send(errorFrame(req.frame.RequestID, err.Error()))
					continue
				}

				// non-blocking send
				req.connctx.send(resp)
			}
		}()
	}

	return wp
}

// submit tries to enqueue request; fails only if queue is full
func (wp *workerpool) submit(req request) bool {
	select {
	case wp.jobs <- req:
		return true
	default:
		return false
	}
}

func (wp *workerpool) stop() {
	close(wp.jobs)
	wp.wg.Wait()
}

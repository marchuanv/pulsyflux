package socket

import (
	"sync"
	"time"
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
				req.cancel() // always cancel after processing

				if err != nil {
					req.connctx.send(errorFrame(req.frame.RequestID, err.Error()))
					continue
				}
				req.connctx.send(resp)
			}
		}()
	}

	return wp
}

func (wp *workerpool) submit(req request) bool {
	select {
	case wp.jobs <- req:
		return true
	case <-time.After(workerQueueTimeout):
		req.cancel() // cancel if submission fails
		return false
	case <-req.ctx.Done():
		req.cancel()
		return false
	}
}

func (wp *workerpool) stop() {
	close(wp.jobs)
	wp.wg.Wait()
}

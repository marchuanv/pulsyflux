package socket

import (
	"sync"
	"time"
)

type requestHandler struct {
	requests      []chan *request // One queue per worker
	wg            sync.WaitGroup
	numWorkers    int
}

func newRequestHandler(noHandlers, handlerQueueSize int, clientRegistry *clientRegistry) *requestHandler {
	rh := &requestHandler{
		requests:   make([]chan *request, noHandlers),
		numWorkers: noHandlers,
	}
	for i := 0; i < noHandlers; i++ {
		rh.requests[i] = make(chan *request, handlerQueueSize/noHandlers)
		rw := newRequestWorker(uint64(i), clientRegistry, rh)
		rh.wg.Add(1)
		go rw.handle(rh.requests[i], &rh.wg)
	}
	return rh
}

func (rh *requestHandler) handle(req *request) bool {
	// Hash RequestID to a specific worker to ensure sequential processing
	workerID := int(req.requestID[0]) % rh.numWorkers
	select {
	case rh.requests[workerID] <- req:
		return true
	case <-time.After(workerQueueTimeout):
		return false
	}
}

func (rh *requestHandler) stop() {
	for _, ch := range rh.requests {
		close(ch)
	}
	rh.wg.Wait()
}

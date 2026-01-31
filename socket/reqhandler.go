package socket

import (
	"sync"
)

type requestHandler struct {
	requests chan *request
	wg       sync.WaitGroup
}

func newRequestHandler(noHandlers, handlerQueueSize int, clientRegistry *clientRegistry) *requestHandler {
	rh := &requestHandler{
		requests: make(chan *request, handlerQueueSize),
	}
	for i := 0; i < noHandlers; i++ {
		rw := newRequestWorker(uint64(i), clientRegistry)
		rh.wg.Add(1)
		go rw.handle(rh.requests, &rh.wg)
	}
	return rh
}

func (rh *requestHandler) handle(req *request) bool {
	select {
	case rh.requests <- req:
		return true
	default:
		return false
	}
}

func (rh *requestHandler) stop() {
	close(rh.requests)
	rh.wg.Wait()
}

package socket

import (
	"context"
	"sync"
	"time"
)

type requestHandler struct {
	requests chan *request
	wg       sync.WaitGroup
}

func newRequestHandler(noHandlers, handlerQueueSize int, registry *clientRegistry) *requestHandler {
	rh := &requestHandler{
		requests: make(chan *request, handlerQueueSize),
	}

	for i := 0; i < noHandlers; i++ {
		workerID := uint64(i)
		rh.wg.Add(1)

		go func(id uint64) {
			defer rh.wg.Done()
			for req := range rh.requests {
				req.consumerID = id

				// Check if canceled before processing
				select {
				case <-req.ctx.Done():
					req.cancel()
					continue
				default:
				}

				if req.isProvider {
					// Send payload to the consumer
					consumerCtx, ok := registry.getConsumer(req.consumerID)
					if ok {
						consumerCtx.send(req.frame) // send data to consumer
					} else {
						// If no consumer connected, fail the request
						req.connctx.send(errorFrame(req.requestID, "consumer"+req.consumerID+" is not available"))
					}
					// Wait for consumer to respond (blocking or via another channel)
					select {
					case resp := <-req.responseChan:
						req.cancel()
						return resp, nil
					case <-time.After(req.timeout):
						req.connctx.send(errorFrame(req.requestID, context.DeadlineExceeded.Error()))
						req.cancel()
						return
					case <-req.ctx.Done():
						req.connctx.send(errorFrame(req.requestID, context.Canceled.Error()))
						req.cancel()
						return
					}
				} else if req.isConsumer {

					// Send processed response back to provider
					providerCtx, ok := registry.getProvider(req.providerID)
					if !ok {
						req.connctx.send(errorFrame(req.requestID, "provider"+req.providerID+" is not available"))
						continue
					}
					providerCtx.send(resp)
					req.cancel()

				}
			}
		}(workerID)
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

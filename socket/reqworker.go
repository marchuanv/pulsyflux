package socket

import (
	"context"
	"sync"
	"time"
)

type requestWorker struct {
	workerID uint64
	registry *clientRegistry
}

func newRequestWorker(workerID uint64, registry *clientRegistry) *requestWorker {
	return &requestWorker{
		workerID: workerID,
		registry: registry,
	}
}

func (rw *requestWorker) handle(reqCh chan *request, wg *sync.WaitGroup) {
	defer wg.Done()

	for req := range reqCh {

		// Skip canceled requests immediately
		select {
		case <-req.ctx.Done():
			req.cancel()
			continue
		default:
		}

		// Check if a peer is available for this role + channel
		if !rw.registry.hasPeerForChannel(req.role, req.channelID) {
			req.connctx.send(newErrorFrame(req.requestID, "no peer available for channel"))
			req.cancel()
			continue
		}

		// Forward the frame to a peer
		peerCtx, ok := rw.registry.getPeerForChannel(req.role, req.channelID)
		if !ok {
			// This should rarely happen if hasPeerForChannel returned true
			req.connctx.send(newErrorFrame(req.requestID, "peer disappeared"))
			req.cancel()
			continue
		}

		if !peerCtx.send(req.frame) {
			req.connctx.send(newErrorFrame(req.requestID, "failed to send frame to peer"))
			req.cancel()
			continue
		}

		// Wait for response, timeout, or cancellation
		select {
		case resp := <-req.resCh:
			req.connctx.send(resp)
			req.cancel()

		case <-time.After(req.timeout):
			req.connctx.send(newErrorFrame(req.requestID, context.DeadlineExceeded.Error()))
			req.cancel()

		case <-req.ctx.Done():
			req.connctx.send(newErrorFrame(req.requestID, context.Canceled.Error()))
			req.cancel()
		}
	}
}

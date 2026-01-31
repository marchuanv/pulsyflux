package socket

import (
	"sync"
)

type requestWorker struct {
	workerID       uint64
	clientRegistry *clientRegistry
}

func newRequestWorker(workerID uint64, clientRegistry *clientRegistry) *requestWorker {
	return &requestWorker{
		workerID:       workerID,
		clientRegistry: clientRegistry,
	}
}

func (rw *requestWorker) handle(reqCh chan *request, wg *sync.WaitGroup) {
	defer wg.Done()

	for req := range reqCh {

		select {
		case <-req.ctx.Done():
			req.cancel()
			continue
		default:
		}

		if !rw.clientRegistry.hasPeerForChannel(req.role, req.channelID) {
			req.connctx.send(newErrorFrame(req.requestID, "no peer available for channel"))
			req.cancel()
			continue
		}

		peerCtx, ok := rw.clientRegistry.getPeerForChannel(req.role, req.channelID)
		if !ok {
			req.connctx.send(newErrorFrame(req.requestID, "peer disappeared"))
			req.cancel()
			continue
		}

		// Forward with original requestID and set forwarded flag
		req.frame.Flags |= FlagForwarded
		req.frame.Payload = req.payload
		if !peerCtx.send(req.frame) {
			req.connctx.send(newErrorFrame(req.requestID, "failed to send to peer"))
		}

		req.cancel()
	}
}

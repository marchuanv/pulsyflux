package socket

import (
	"sync"
)

type requestWorker struct {
	workerID       uint64
	clientRegistry *clientRegistry
	handler        *requestHandler
	routingBuf     [32]byte // Reuse buffer for routing info
}

func newRequestWorker(workerID uint64, clientRegistry *clientRegistry, handler *requestHandler) *requestWorker {
	return &requestWorker{
		workerID:       workerID,
		clientRegistry: clientRegistry,
		handler:        handler,
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

		peerCtx, ok := rw.clientRegistry.getPeerForChannel(req.role, req.channelID)
		if !ok {
			req.connctx.send(newErrorFrame(req.requestID, "peer disappeared"))
			if req.frame.Type == endFrame {
				req.cancel()
			}
			continue
		}

		// Forward frame based on type
		switch req.frame.Type {
		case startFrame:
			// Reuse worker's routing buffer
			copy(rw.routingBuf[0:16], req.clientID[:])
			copy(rw.routingBuf[16:32], req.channelID[:])
			req.frame.Payload = rw.routingBuf[:]
			if !peerCtx.send(req.frame) {
				req.connctx.send(newErrorFrame(req.requestID, "failed to send start to peer"))
				req.cancel()
				continue
			}

		case chunkFrame:
			if !peerCtx.send(req.frame) {
				req.connctx.send(newErrorFrame(req.requestID, "failed to send chunk to peer"))
				continue
			}

		case endFrame:
			if !peerCtx.send(req.frame) {
				req.connctx.send(newErrorFrame(req.requestID, "failed to send end to peer"))
			}
			req.cancel()
		}
	}
}

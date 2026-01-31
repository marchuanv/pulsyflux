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

		peerCtx, ok := rw.clientRegistry.getPeerForChannel(req.role, req.channelID)
		if !ok {
			req.connctx.send(newErrorFrame(req.requestID, "peer disappeared"))
			if req.frame.Type == EndFrame {
				req.cancel()
			}
			continue
		}

		// Forward frame based on type
		switch req.frame.Type {
		case StartFrame:
			// Send only routing info in StartFrame, not the original metadata
			routingInfo := make([]byte, 32)
			copy(routingInfo[0:16], req.clientID[:])
			copy(routingInfo[16:32], req.channelID[:])

			startFrame := *req.frame
			startFrame.Payload = routingInfo
			if !peerCtx.send(&startFrame) {
				req.connctx.send(newErrorFrame(req.requestID, "failed to send start to peer"))
				req.cancel()
				continue
			}

		case ChunkFrame:
			if !peerCtx.send(req.frame) {
				req.connctx.send(newErrorFrame(req.requestID, "failed to send chunk to peer"))
				continue
			}

		case EndFrame:
			if !peerCtx.send(req.frame) {
				req.connctx.send(newErrorFrame(req.requestID, "failed to send end to peer"))
			}
			req.cancel()
		}
	}
}

package socket

import (
	"log"
	"sync"
)

type requestWorker struct {
	workerID   uint64
	handler    *requestHandler
	routingBuf [32]byte // Reuse buffer for routing info
	peers      *peers
}

func newRequestWorker(workerID uint64, handler *requestHandler, peers *peers) *requestWorker {
	return &requestWorker{
		workerID: workerID,
		handler:  handler,
		peers:    peers,
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
		peerCtx, ok := rw.peers.get(req.peerClientID)
		if !ok {
			log.Printf("[Worker] ERROR: Peer %s not found for request %s from client %s", req.peerClientID, req.requestID, req.clientID)
			req.connctx.enqueue(newErrorFrame(req.requestID, req.clientID, "no peer client available"))
			req.cancel()
			continue
		}
		if !peerCtx.enqueue(req.frame) {
			req.connctx.enqueue(newErrorFrame(req.requestID, req.clientID, "failed to send frame to peer"))
		}
		req.cancel()
	}
}

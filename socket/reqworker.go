package socket

import (
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
		peer, ok := rw.peers.get(req.peerClientID)
		if !ok {
			req.connctx.enqueue(newErrorFrame(req.requestID, req.clientID, "no peer client available", flagPeerNotAvailable))
			req.cancel()
			continue
		}
		
		// Route frame to peer
		if !peer.connctx.enqueue(req.frame) {
			req.connctx.enqueue(newErrorFrame(req.requestID, req.clientID, "failed to send frame to peer", 0))
		}
		req.cancel()
	}
}

func frameTypeName(t byte) string {
	switch t {
	case errorFrame:
		return "ERROR"
	case startFrame:
		return "START"
	case chunkFrame:
		return "CHUNK"
	case endFrame:
		return "END"
	default:
		return "UNKNOWN"
	}
}

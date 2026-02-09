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
		peer, ok := rw.peers.get(req.peerClientID)
		if !ok {
			log.Printf("[Worker] ERROR: Peer %s not found for request %s from client %s", req.peerClientID, req.requestID, req.clientID)
			req.connctx.enqueue(newErrorFrame(req.requestID, req.clientID, "no peer client available", flagPeerNotAvailable))
			req.cancel()
			continue
		}
		
		senderPeer, senderOk := rw.peers.get(req.clientID)
		if senderOk && senderPeer.role != peer.role {
			// Only map if roles are different (provider <-> consumer)
			if mappedReqID, found := senderPeer.mapper.getMappedRequestID(req.requestID); found {
				// Request ID has a mapping - translate it back
				req.frame.RequestID = mappedReqID
				log.Printf("[Worker] Mapped request ID %s -> %s", req.requestID, mappedReqID)
			} else if req.frame.Type == startFrame && req.frame.Flags == flagNone {
				// New START frame - check if peer has a pending request to map to
				// For now, just forward it and let the receiver handle correlation
				log.Printf("[Worker] Forwarding START frame with request ID %s", req.requestID)
			}
		}
		
		if !peer.connctx.enqueue(req.frame) {
			req.connctx.enqueue(newErrorFrame(req.requestID, req.clientID, "failed to send frame to peer", 0))
		}
		log.Printf("[Worker] Routed request %s from client %s to peer %s", req.requestID, req.clientID, req.peerClientID)
		req.cancel()
	}
}

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
		if senderOk {
			if mappedReqID, found := senderPeer.mapper.getMappedRequestID(req.requestID); found {
				log.Printf("[Worker] Response: Mapping request ID %s -> %s", req.requestID, mappedReqID)
				req.frame.RequestID = mappedReqID
				if req.frame.Type == endFrame {
					senderPeer.mapper.removeMapping(req.requestID)
				}
			}
		}
		
		if !peer.connctx.enqueue(req.frame) {
			req.connctx.enqueue(newErrorFrame(req.requestID, req.clientID, "failed to send frame to peer", 0))
		}
		log.Printf("[Worker] Routed request %s from client %s to peer %s", req.requestID, req.clientID, req.peerClientID)
		req.cancel()
	}
}

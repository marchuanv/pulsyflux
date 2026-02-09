package socket

import (
	"sync"

	"github.com/google/uuid"
)

type peer struct {
	clientID  uuid.UUID
	channelID uuid.UUID
	peerID    uuid.UUID
	connctx   *connctx
	mapper    *requestMapper
}

type peers struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]*peer
}

func newPeers() *peers {
	return &peers{
		clients: make(map[uuid.UUID]*peer),
	}
}

func (r *peers) get(clientID uuid.UUID) (*peer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	peer, ok := r.clients[clientID]
	if !ok {
		return nil, false
	}
	return peer, true
}

func (r *peers) set(clientID uuid.UUID, ctx *connctx, channelId uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[clientID] = &peer{
		clientID:  clientID,
		channelID: channelId,
		peerID:    uuid.Nil,
		connctx:   ctx,
		mapper:    newRequestMapper(),
	}
}

func (r *peers) delete(clientID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.clients[clientID]
	if ok {
		delete(r.clients, clientID)
	}
}

func (r *peers) pair(clientID uuid.UUID, channelId uuid.UUID) *peer {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, peer := range r.clients {
		if peer.channelID == channelId && peer.clientID != clientID {
			return peer
		}
	}
	return nil
}

func (r *peers) setPeer(clientID uuid.UUID, peerID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.clients[clientID]; ok {
		p.peerID = peerID
	}
}

package socket

import (
	"sync"

	"github.com/google/uuid"
)

type peer struct {
	clientID  uuid.UUID
	channelID uuid.UUID
	role      clientRole
	connctx   *connctx
}

type peers struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]*peer // channelID -> clientID -> connctx
}

func newPeers() *peers {
	return &peers{
		clients: make(map[uuid.UUID]*peer),
	}
}

func (r *peers) get(clientID uuid.UUID) (*connctx, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ctx, ok := r.clients[clientID]
	if !ok {
		return nil, false
	}
	return ctx.connctx, true
}

func (r *peers) set(clientID uuid.UUID, ctx *connctx, role clientRole, channelId uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[clientID] = &peer{
		clientID:  clientID,
		role:      role,
		channelID: channelId,
		connctx:   ctx,
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

func (r *peers) pair(clientID uuid.UUID, role clientRole, channelId uuid.UUID) *peer {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, peer := range r.clients {
		if peer.channelID == channelId && peer.role != role && peer.clientID != clientID {
			return peer
		}
	}
	return nil
}

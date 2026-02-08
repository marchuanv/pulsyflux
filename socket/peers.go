package socket

import (
	"sync"

	"github.com/google/uuid"
)

type peers struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]*connctx // channelID -> clientID -> connctx
}

func newPeers() *peers {
	return &peers{
		clients: make(map[uuid.UUID]*connctx),
	}
}

func (r *peers) get(clientID uuid.UUID) (*connctx, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ctx, ok := r.clients[clientID]
	return ctx, ok
}

// Registers a client without checking peers
func (r *peers) set(clientID uuid.UUID, ctx *connctx) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[clientID] = ctx
}

// Removes a client
func (r *peers) delete(clientID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.clients[clientID]
	if ok {
		delete(r.clients, clientID)
	}
}

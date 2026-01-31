package socket

import (
	"sync"

	"github.com/google/uuid"
)

type ClientRole byte

const (
	RoleConsumer ClientRole = 0x01
	RoleProvider ClientRole = 0x02
)

type clientRegistry struct {
	mu             sync.RWMutex
	consumers      map[uuid.UUID]map[uuid.UUID]*connctx // channelID -> clientID -> connctx
	providers      map[uuid.UUID]map[uuid.UUID]*connctx // channelID -> clientID -> connctx
	peerToOriginal map[uuid.UUID]uuid.UUID              // peer's requestID -> original requestID
	originalToPeer map[uuid.UUID]uuid.UUID              // original requestID -> peer's requestID
}

func newClientRegistry() *clientRegistry {
	return &clientRegistry{
		consumers:      make(map[uuid.UUID]map[uuid.UUID]*connctx),
		providers:      make(map[uuid.UUID]map[uuid.UUID]*connctx),
		peerToOriginal: make(map[uuid.UUID]uuid.UUID),
		originalToPeer: make(map[uuid.UUID]uuid.UUID),
	}
}

func (r *clientRegistry) trackRequest(originalID, peerID uuid.UUID) {
	r.mu.Lock()
	r.peerToOriginal[peerID] = originalID
	r.originalToPeer[originalID] = peerID
	r.mu.Unlock()
}

func (r *clientRegistry) getOriginalRequestID(peerID uuid.UUID) (uuid.UUID, bool) {
	r.mu.RLock()
	originalID, ok := r.peerToOriginal[peerID]
	r.mu.RUnlock()
	return originalID, ok
}

func (r *clientRegistry) untrackRequest(originalID uuid.UUID) {
	r.mu.Lock()
	if peerID, ok := r.originalToPeer[originalID]; ok {
		delete(r.peerToOriginal, peerID)
		delete(r.originalToPeer, originalID)
	}
	r.mu.Unlock()
}

func (r *clientRegistry) getClient(role ClientRole, channelID, clientID uuid.UUID) (*connctx, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	switch role {
	case RoleConsumer:
		if clients, ok := r.consumers[channelID]; ok {
			if ctx, ok := clients[clientID]; ok {
				return ctx, true
			}
		}
	case RoleProvider:
		if clients, ok := r.providers[channelID]; ok {
			if ctx, ok := clients[clientID]; ok {
				return ctx, true
			}
		}
	}
	return nil, false
}

func (r *clientRegistry) hasClient(role ClientRole, channelID, clientID uuid.UUID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	switch role {
	case RoleConsumer:
		if clients, ok := r.consumers[channelID]; ok {
			_, exists := clients[clientID]
			return exists
		}
	case RoleProvider:
		if clients, ok := r.providers[channelID]; ok {
			_, exists := clients[clientID]
			return exists
		}
	}

	return false
}

// Registers a client without checking peers
func (r *clientRegistry) addClient(role ClientRole, channelID, clientID uuid.UUID, ctx *connctx) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch role {
	case RoleConsumer:
		if r.consumers[channelID] == nil {
			r.consumers[channelID] = make(map[uuid.UUID]*connctx)
		}
		r.consumers[channelID][clientID] = ctx
	case RoleProvider:
		if r.providers[channelID] == nil {
			r.providers[channelID] = make(map[uuid.UUID]*connctx)
		}
		r.providers[channelID][clientID] = ctx
	}
}

// Removes a client
func (r *clientRegistry) removeClient(role ClientRole, channelID, clientID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var clients map[uuid.UUID]*connctx
	switch role {
	case RoleConsumer:
		clients = r.consumers[channelID]
		if clients != nil {
			delete(clients, clientID)
			if len(clients) == 0 {
				delete(r.consumers, channelID)
			}
		}
	case RoleProvider:
		clients = r.providers[channelID]
		if clients != nil {
			delete(clients, clientID)
			if len(clients) == 0 {
				delete(r.providers, channelID)
			}
		}
	}
}

func (r *clientRegistry) hasPeerForChannel(role ClientRole, channelID uuid.UUID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	switch role {
	case RoleConsumer:
		// consumer needs a provider
		return len(r.providers[channelID]) > 0
	case RoleProvider:
		// provider needs a consumer
		return len(r.consumers[channelID]) > 0
	default:
		return false
	}
}

func (r *clientRegistry) getPeerForChannel(role ClientRole, channelID uuid.UUID) (*connctx, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var peers map[uuid.UUID]*connctx
	switch role {
	case RoleConsumer:
		peers = r.providers[channelID] // consumer wants a provider
	case RoleProvider:
		peers = r.consumers[channelID] // provider wants a consumer
	default:
		return nil, false
	}

	for _, ctx := range peers {
		return ctx, true // return the first available peer
	}

	return nil, false
}

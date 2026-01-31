package socket

import (
	"sync"

	"github.com/google/uuid"
)

type ClientRole byte

const (
	RoleConsumer ClientRole = 0x01 // single byte: 0x01
	RoleProvider ClientRole = 0x02 // single byte: 0x01
)

type clientRegistry struct {
	mu        sync.RWMutex
	consumers map[uuid.UUID]map[uuid.UUID]*connctx // channelID -> clientID -> connctx
	providers map[uuid.UUID]map[uuid.UUID]*connctx // channelID -> clientID -> connctx
}

func newClientRegistry() *clientRegistry {
	return &clientRegistry{
		consumers: make(map[uuid.UUID]map[uuid.UUID]*connctx),
		providers: make(map[uuid.UUID]map[uuid.UUID]*connctx),
	}
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

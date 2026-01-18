package socket

import (
	"sync"
)

// Manages active clients
type clientRegistry struct {
	mu        sync.RWMutex
	providers map[uint64]*connctx
	consumers map[uint64]*connctx
}

func newClientRegistry() *clientRegistry {
	return &clientRegistry{
		providers: make(map[uint64]*connctx),
		consumers: make(map[uint64]*connctx),
	}
}

func (r *clientRegistry) addProvider(id uint64, ctx *connctx) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[id] = ctx
}

func (r *clientRegistry) addConsumer(id uint64, ctx *connctx) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.consumers[id] = ctx
}

func (r *clientRegistry) getProvider(id uint64) (*connctx, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ctx, ok := r.providers[id]
	return ctx, ok
}

func (r *clientRegistry) getConsumer(id uint64) (*connctx, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ctx, ok := r.consumers[id]
	return ctx, ok
}

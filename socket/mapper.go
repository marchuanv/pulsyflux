package socket

import (
	"sync"

	"github.com/google/uuid"
)

type requestMapper struct {
	mu      sync.RWMutex
	mapping map[uuid.UUID]uuid.UUID // Bidirectional mapping of request IDs
	pending uuid.UUID                // Pending provider request ID waiting for consumer response
}

func newRequestMapper() *requestMapper {
	return &requestMapper{
		mapping: make(map[uuid.UUID]uuid.UUID),
	}
}

func (rm *requestMapper) setPending(reqID uuid.UUID) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.pending = reqID
}

func (rm *requestMapper) getPending() (uuid.UUID, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	if rm.pending == uuid.Nil {
		return uuid.Nil, false
	}
	return rm.pending, true
}

func (rm *requestMapper) mapRequest(reqID1, reqID2 uuid.UUID) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.mapping[reqID1] = reqID2
	rm.mapping[reqID2] = reqID1
	rm.pending = uuid.Nil // Clear pending after mapping
}

func (rm *requestMapper) getMappedRequestID(reqID uuid.UUID) (uuid.UUID, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	mappedReqID, ok := rm.mapping[reqID]
	return mappedReqID, ok
}

func (rm *requestMapper) removeMapping(reqID uuid.UUID) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if mappedReqID, ok := rm.mapping[reqID]; ok {
		delete(rm.mapping, mappedReqID)
	}
	delete(rm.mapping, reqID)
}

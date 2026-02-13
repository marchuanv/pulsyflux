package socket

import (
	"sync"

	"github.com/google/uuid"
)

type peer struct {
	clientID       uuid.UUID
	channelID      uuid.UUID
	connctx        *connctx
	mapper         *requestMapper
	handshakeReqID uuid.UUID
}

type peers struct {
	mu       sync.RWMutex
	clients  map[uuid.UUID]*peer
	channels map[uuid.UUID]map[uuid.UUID]*peer
}

func newPeers() *peers {
	return &peers{
		clients:  make(map[uuid.UUID]*peer),
		channels: make(map[uuid.UUID]map[uuid.UUID]*peer),
	}
}

func (r *peers) get(clientID uuid.UUID) (*peer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	peer, ok := r.clients[clientID]
	return peer, ok
}

func (r *peers) set(clientID uuid.UUID, ctx *connctx, channelID uuid.UUID, handshakeReqID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := &peer{
		clientID:       clientID,
		channelID:      channelID,
		connctx:        ctx,
		mapper:         newRequestMapper(),
		handshakeReqID: handshakeReqID,
	}
	r.clients[clientID] = p
	if r.channels[channelID] == nil {
		r.channels[channelID] = make(map[uuid.UUID]*peer)
	}
	r.channels[channelID][clientID] = p
}

func (r *peers) delete(clientID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.clients[clientID]; ok {
		delete(r.clients, clientID)
		if ch := r.channels[p.channelID]; ch != nil {
			delete(ch, clientID)
			if len(ch) == 0 {
				delete(r.channels, p.channelID)
			}
		}
	}
}

func (r *peers) getChannelPeers(channelID uuid.UUID, excludeClientID uuid.UUID) []*peer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ch := r.channels[channelID]
	if ch == nil {
		return nil
	}
	peers := make([]*peer, 0, len(ch)-1)
	for id, p := range ch {
		if id != excludeClientID {
			peers = append(peers, p)
		}
	}
	return peers
}

func (r *peers) pair(clientID uuid.UUID, channelID uuid.UUID) *peer {
	peers := r.getChannelPeers(channelID, clientID)
	if len(peers) > 0 {
		return peers[0]
	}
	return nil
}

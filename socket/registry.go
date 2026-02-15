package socket

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type ackCollector struct {
	frameID      uuid.UUID
	expectedAcks int
	receivedAcks int
	done         chan struct{}
	closed       bool
	mu           sync.Mutex
}

type clientEntry struct {
	clientID      uuid.UUID
	channelID     uuid.UUID
	connctx       *connctx
	requestQueue  chan *frame
	responseQueue chan *frame
}

type registry struct {
	mu          sync.RWMutex
	clients     map[uuid.UUID]*clientEntry
	channels    map[uuid.UUID]map[uuid.UUID]*clientEntry
	pendingAcks map[uuid.UUID]*ackCollector
	channelNotify map[uuid.UUID][]chan struct{}
}

func newRegistry() *registry {
	return &registry{
		clients:       make(map[uuid.UUID]*clientEntry),
		channels:      make(map[uuid.UUID]map[uuid.UUID]*clientEntry),
		pendingAcks:   make(map[uuid.UUID]*ackCollector),
		channelNotify: make(map[uuid.UUID][]chan struct{}),
	}
}

func (r *registry) register(clientID, channelID uuid.UUID, ctx *connctx) *clientEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := &clientEntry{
		clientID:      clientID,
		channelID:     channelID,
		connctx:       ctx,
		requestQueue:  make(chan *frame, 1024),
		responseQueue: make(chan *frame, 1024),
	}

	r.clients[clientID] = entry
	if r.channels[channelID] == nil {
		r.channels[channelID] = make(map[uuid.UUID]*clientEntry)
	}
	r.channels[channelID][clientID] = entry

	// Notify any waiting senders that a new receiver is available
	if waiters := r.channelNotify[channelID]; len(waiters) > 0 {
		for _, ch := range waiters {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
		delete(r.channelNotify, channelID)
	}

	go entry.processResponses()
	go entry.processRequests()
	return entry
}

func (r *registry) get(clientID uuid.UUID) (*clientEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clients[clientID], r.clients[clientID] != nil
}

func (r *registry) unregister(clientID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.clients[clientID]; ok {
		close(entry.requestQueue)
		close(entry.responseQueue)
		delete(r.clients, clientID)

		if ch := r.channels[entry.channelID]; ch != nil {
			delete(ch, clientID)
			if len(ch) == 0 {
				delete(r.channels, entry.channelID)
			}
		}
	}
}

func (r *registry) waitForReceivers(channelID, excludeClientID uuid.UUID, timeoutMs uint64) []*clientEntry {
	r.mu.Lock()
	peers := r.getChannelPeersLocked(channelID, excludeClientID)
	if len(peers) > 0 {
		r.mu.Unlock()
		return peers
	}
	
	// No receivers yet, register for notification
	notifyCh := make(chan struct{}, 1)
	r.channelNotify[channelID] = append(r.channelNotify[channelID], notifyCh)
	r.mu.Unlock()
	
	timeout := time.After(time.Duration(timeoutMs) * time.Millisecond)
	for {
		select {
		case <-notifyCh:
			r.mu.Lock()
			peers := r.getChannelPeersLocked(channelID, excludeClientID)
			r.mu.Unlock()
			if len(peers) > 0 {
				return peers
			}
		case <-timeout:
			return nil
		}
	}
}

func (r *registry) getChannelPeersLocked(channelID, excludeClientID uuid.UUID) []*clientEntry {
	ch := r.channels[channelID]
	if ch == nil {
		return nil
	}

	peers := make([]*clientEntry, 0, len(ch)-1)
	for id, entry := range ch {
		if id != excludeClientID {
			peers = append(peers, entry)
		}
	}
	return peers
}

func (e *clientEntry) enqueueRequest(f *frame) {
	select {
	case e.requestQueue <- f:
	default:
		putFrame(f)
	}
}

func (e *clientEntry) enqueueResponse(f *frame) {
	select {
	case e.responseQueue <- f:
	default:
		putFrame(f)
	}
}

func (e *clientEntry) processResponses() {
	for f := range e.responseQueue {
		select {
		case e.connctx.writes <- f:
		case <-e.connctx.closed:
			putFrame(f)
			return
		}
	}
}

func (e *clientEntry) processRequests() {
	for f := range e.requestQueue {
		select {
		case e.connctx.writes <- f:
		case <-e.connctx.closed:
			putFrame(f)
			return
		}
	}
}

func (r *registry) createAckCollector(frameID uuid.UUID, expectedAcks int) *ackCollector {
	r.mu.Lock()
	defer r.mu.Unlock()
	collector := &ackCollector{
		frameID:      frameID,
		expectedAcks: expectedAcks,
		done:         make(chan struct{}),
	}
	r.pendingAcks[frameID] = collector
	return collector
}

func (r *registry) recordAck(frameID uuid.UUID) {
	r.mu.Lock()
	collector, ok := r.pendingAcks[frameID]
	r.mu.Unlock()
	if !ok {
		return
	}
	collector.mu.Lock()
	collector.receivedAcks++
	if collector.receivedAcks == collector.expectedAcks && !collector.closed {
		collector.closed = true
		close(collector.done)
	}
	collector.mu.Unlock()
}

func (r *registry) removeAckCollector(frameID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.pendingAcks, frameID)
}

package socket

import (
	"log"
	"sync"

	"github.com/google/uuid"
)

type ackCollector struct {
	frameID      uuid.UUID
	requestID    uuid.UUID
	senderID     uuid.UUID
	frameType    byte
	expectedAcks int
	receivedAcks int
	done         chan struct{}
	mu           sync.Mutex
}

type clientEntry struct {
	clientID       uuid.UUID
	channelID      uuid.UUID
	connctx        *connctx
	requestQueue   chan *frame
	responseQueue  chan *frame
	pendingReceive chan *frame
}

type registry struct {
	mu          sync.RWMutex
	clients     map[uuid.UUID]*clientEntry
	channels    map[uuid.UUID]map[uuid.UUID]*clientEntry
	pendingAcks map[uuid.UUID]*ackCollector
}

func newRegistry() *registry {
	return &registry{
		clients:     make(map[uuid.UUID]*clientEntry),
		channels:    make(map[uuid.UUID]map[uuid.UUID]*clientEntry),
		pendingAcks: make(map[uuid.UUID]*ackCollector),
	}
}

func (r *registry) register(clientID, channelID uuid.UUID, ctx *connctx) *clientEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	log.Printf("[Registry] Registering client %s on channel %s", clientID.String()[:8], channelID.String()[:8])

	entry := &clientEntry{
		clientID:       clientID,
		channelID:      channelID,
		connctx:        ctx,
		requestQueue:   make(chan *frame, 1024),
		responseQueue:  make(chan *frame, 1024),
		pendingReceive: make(chan *frame, 1024),
	}

	r.clients[clientID] = entry
	if r.channels[channelID] == nil {
		r.channels[channelID] = make(map[uuid.UUID]*clientEntry)
	}
	r.channels[channelID][clientID] = entry

	go entry.processResponses()
	go entry.processRequests()
	return entry
}

func (r *registry) get(clientID uuid.UUID) (*clientEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.clients[clientID]
	return entry, ok
}

func (r *registry) unregister(clientID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.clients[clientID]; ok {
		close(entry.requestQueue)
		close(entry.responseQueue)
		close(entry.pendingReceive)
		delete(r.clients, clientID)

		if ch := r.channels[entry.channelID]; ch != nil {
			delete(ch, clientID)
			if len(ch) == 0 {
				delete(r.channels, entry.channelID)
			}
		}
	}
}

func (r *registry) getChannelPeers(channelID, excludeClientID uuid.UUID) []*clientEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

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

func (e *clientEntry) enqueueRequest(f *frame) bool {
	select {
	case e.requestQueue <- f:
		return true
	default:
		return false
	}
}

func (e *clientEntry) dequeueRequest() (*frame, bool) {
	select {
	case f, ok := <-e.requestQueue:
		return f, ok
	default:
		return nil, false
	}
}

func (e *clientEntry) enqueueResponse(f *frame) bool {
	select {
	case e.responseQueue <- f:
		return true
	default:
		return false
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

func (r *registry) createAckCollector(frameID, requestID, senderID uuid.UUID, frameType byte, expectedAcks int) *ackCollector {
	r.mu.Lock()
	defer r.mu.Unlock()
	collector := &ackCollector{
		frameID:      frameID,
		requestID:    requestID,
		senderID:     senderID,
		frameType:    frameType,
		expectedAcks: expectedAcks,
		receivedAcks: 0,
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
	if collector.receivedAcks == collector.expectedAcks {
		close(collector.done)
	}
	collector.mu.Unlock()
}

func (r *registry) removeAckCollector(frameID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.pendingAcks, frameID)
}

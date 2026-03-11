# Design Document: Message Broker Delivery Guarantees

## Overview

This design extends the existing PulsyFlux broker to support reliable message delivery and state synchronization for distributed applications requiring strong consistency guarantees. The system implements deterministic message ordering with at-least-once delivery semantics, enabling 2-10 concurrent clients with precise message ordering and state consistency.

The design introduces a comprehensive message broker architecture with:
1. **Message Delivery System**: Reliable at-least-once delivery with sequence numbers and acknowledgments
2. **State Synchronization**: Snapshot-based state distribution with delta encoding
3. **Reconnection Management**: Token-based session recovery with message buffering
4. **Deterministic Ordering**: Tick-based message batching with consistent ordering
5. **Message Validation**: Server-side validation pipeline with rate limiting
6. **Message Persistence**: Durable storage for messages with recovery support

### Design Goals

- Support 2-10 concurrent clients with 10-30 messages/second per client
- Maintain deterministic state across all clients (lockstep synchronization)
- Enable seamless reconnection after network interruptions
- Minimize bandwidth with delta encoding for state updates
- Provide low-latency message delivery (<50ms p99)
- Support high throughput (300 messages/second total)
- Enable server-side message validation and authorization
- Maintain backward compatibility with existing broker API

### Messaging Model

This design implements a hybrid messaging model:
- **Messages**: Lockstep with deterministic ordering (at-least-once delivery)
- **State Updates**: Snapshot + delta for new/reconnecting clients
- **Non-Critical Updates**: Optional at-most-once for high-frequency updates

### Use Cases

This broker is designed for demanding real-time applications including:
- **RTS Games**: Lockstep networking with deterministic command execution
- **Financial Trading**: Order execution with guaranteed delivery and ordering
- **IoT Command & Control**: Reliable command delivery to device fleets
- **Collaborative Applications**: Synchronized state across multiple users
- **Distributed Simulations**: Deterministic execution across multiple nodes

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Broker Server                                │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                         Session                               │   │
│  │                                                               │   │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐    │   │
│  │  │  Sequence   │  │ Deterministic│  │     Tick        │    │   │
│  │  │  Manager    │  │   Ordering   │  │    Engine       │    │   │
│  │  └─────────────┘  └──────────────┘  └─────────────────┘    │   │
│  │                                                               │   │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐    │   │
│  │  │   Message   │  │    State     │  │  Reconnection   │    │   │
│  │  │  Validator  │  │  Snapshot    │  │     Manager     │    │   │
│  │  └─────────────┘  └──────────────┘  └─────────────────┘    │   │
│  │                                                               │   │
│  └───────────────────────────┬───────────────────────────────────┘   │
│                              │                                       │
│  ┌───────────────────────────┴───────────────────────────────────┐  │
│  │                                                                 │  │
│  │  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐    │  │
│  │  │   Message    │    │Acknowledgment│    │ Idempotency  │    │  │
│  │  │    Store     │    │   Manager    │    │   Manager    │    │  │
│  │  └──────────────┘    └──────────────┘    └──────────────┘    │  │
│  │                                                                 │  │
│  └─────────────────────────────────────────────────────────────────┘  │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

### Component Interactions

**Message Flow (Lockstep):**
```
Client → Broker → Sequence Manager → Message Validator
                ↓                              ↓
           Message Store                   (valid?)
                ↓                              ↓
           Tick Engine ← ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┘
                ↓
           Deterministic Ordering
                ↓
           Broadcast to All Clients
                ↓
           Ack Manager (track delivery)
                ↓
           Client Ack → Remove from Store
```

**State Synchronization Flow:**
```
State Changes → Delta Generator → Compare with Previous
                          ↓
                    Size Check (>80% of snapshot?)
                          ↓
                    ┌─────┴─────┐
                    │           │
              Delta Update   Full Snapshot
                    │           │
                    └─────┬─────┘
                          ↓
                    Broadcast to Clients
```

**Reconnection Flow:**
```
Client Disconnect → Retain Session State (60s)
                          ↓
                    Buffer Messages (max 1000)
                          ↓
Client Reconnect → Validate Token
                          ↓
                    Send Latest Snapshot
                          ↓
                    Send Buffered Messages
                          ↓
                    Resume Normal Operation
```


## Components and Interfaces

### Session

A session represents a single instance with multiple clients:

```go
type Session struct {
    ID                uuid.UUID
    Config            *SessionConfig
    Clients           map[uuid.UUID]*Client
    SequenceManagers  map[uuid.UUID]*SequenceManager  // per-client sequences
    MessageStore      MessageStore
    AckManager        AckManager
    IdempotencyMgr    IdempotencyManager
    SnapshotManager   *SnapshotManager
    TickEngine        *TickEngine
    MessageValidator  MessageValidator
    ReconnectionMgr   *ReconnectionManager
    Metrics           *SessionMetrics
    mu                sync.RWMutex
}

type SessionConfig struct {
    TickInterval         time.Duration  // 50ms - 500ms
    AckTimeout           time.Duration  // default: 500ms
    MaxDeliveryAttempts  int            // default: 5
    SnapshotInterval     time.Duration  // default: 10s
    SnapshotRetention    int            // default: 3
    ReconnectTimeout     time.Duration  // 30s - 600s
    MaxBufferedMessages  int            // default: 1000
    RateLimitPerClient   int            // default: 100 msg/s
    IdempotencyRetention time.Duration  // default: 60s
}

type Client struct {
    ID                uuid.UUID
    Connection        *tcpconn.Connection
    ReconnectionToken string
    LastSeenTime      time.Time
    Disconnected      bool
    BufferedMessages  []*Message
    mu                sync.RWMutex
}
```

### Message Structure

Messages are the primary type for client actions:

```go
type Message struct {
    ID               uuid.UUID
    SessionID        uuid.UUID
    ClientID         uuid.UUID
    SequenceNumber   uint64
    IdempotencyKey   string
    Payload          []byte
    Timestamp        time.Time
    DeliveryAttempts int
    Acknowledged     bool
}

type MessageEnvelope struct {
    MessageID      string `json:"message_id"`
    ClientID       string `json:"client_id"`
    SequenceNumber uint64 `json:"sequence_number"`
    IdempotencyKey string `json:"idempotency_key"`
    Payload        []byte `json:"payload"`
    Timestamp      int64  `json:"timestamp"`
}
```

### Sequence Manager

Manages monotonically increasing sequence numbers per client:

```go
type SequenceManager struct {
    clientID       uuid.UUID
    nextSequence   atomic.Uint64
    mu             sync.Mutex
}

func (sm *SequenceManager) Next() uint64 {
    return sm.nextSequence.Add(1)
}

func (sm *SequenceManager) Current() uint64 {
    return sm.nextSequence.Load()
}

func (sm *SequenceManager) Reset(value uint64) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    sm.nextSequence.Store(value)
}
```

### Message Store Interface

Persistent storage for messages with recovery support:

```go
type MessageStore interface {
    // Store persists a message atomically
    Store(msg *Message) error
    
    // Get retrieves a message by ID
    Get(messageID uuid.UUID) (*Message, error)
    
    // Delete removes a message from storage
    Delete(messageID uuid.UUID) error
    
    // GetUnacknowledged returns unacked messages for a session
    GetUnacknowledged(sessionID uuid.UUID) ([]*Message, error)
    
    // GetBySequenceRange retrieves messages in a sequence range
    GetBySequenceRange(sessionID, clientID uuid.UUID, start, end uint64) ([]*Message, error)
    
    // UpdateDeliveryAttempt increments delivery attempt count
    UpdateDeliveryAttempt(messageID uuid.UUID) error
    
    // MarkAcknowledged marks a message as acknowledged
    MarkAcknowledged(messageID uuid.UUID) error
    
    // GetHistorical retrieves messages for replay (24h retention)
    GetHistorical(sessionID uuid.UUID, startTime, endTime time.Time) ([]*Message, error)
    
    // Close cleanly shuts down the store
    Close() error
}
```

**Implementation:** Initial implementation uses an in-memory map with mutex protection. Future versions can add disk-based persistence (BadgerDB, BoltDB) for durability across broker restarts.

### Acknowledgment Manager

Tracks message delivery and handles timeouts:

```go
type AckManager interface {
    // TrackDelivery registers a message as pending acknowledgment
    TrackDelivery(messageID uuid.UUID, timeout time.Duration) error
    
    // Acknowledge marks a message as successfully delivered
    Acknowledge(messageID uuid.UUID) error
    
    // GetTimedOut returns messages that have exceeded their ack timeout
    GetTimedOut() []uuid.UUID
    
    // Stop shuts down the manager
    Stop()
}

type pendingAck struct {
    messageID  uuid.UUID
    deadline   time.Time
    timerChan  <-chan time.Time
}
```

**Implementation:** Uses a map of pending acknowledgments with timer goroutines. A background worker periodically checks for timed-out messages and triggers redelivery.

### Idempotency Manager

Handles deduplication for exactly-once semantics:

```go
type IdempotencyManager interface {
    // CheckAndStore checks if a key exists; if not, stores it atomically
    // Returns true if this is a new key (not a duplicate)
    CheckAndStore(key string, sessionID uuid.UUID) (bool, error)
    
    // Remove deletes an idempotency key
    Remove(key string) error
    
    // Cleanup removes expired keys based on retention policy
    Cleanup(retention time.Duration) error
}

type idempotencyRecord struct {
    key       string
    sessionID uuid.UUID
    createdAt time.Time
}
```

**Key Format:** When not provided by client, generated as `{clientID}:{sequenceNumber}`


### Tick Engine (Deterministic Ordering)

Implements lockstep synchronization with tick-based message batching:

```go
type TickEngine struct {
    session       *Session
    tickInterval  time.Duration
    currentTick   atomic.Uint64
    messageBuffer map[uuid.UUID][]*Message  // per-client message buffers
    ticker        *time.Ticker
    stopChan      chan struct{}
    mu            sync.RWMutex
}

func (te *TickEngine) Start() {
    te.ticker = time.NewTicker(te.tickInterval)
    go func() {
        for {
            select {
            case <-te.ticker.C:
                te.processTick()
            case <-te.stopChan:
                te.ticker.Stop()
                return
            }
        }
    }()
}

func (te *TickEngine) processTick() {
    te.mu.Lock()
    defer te.mu.Unlock()
    
    // Collect messages from all clients
    messages := te.collectMessages()
    
    // Sort deterministically
    sort.Slice(messages, func(i, j int) bool {
        if messages[i].SequenceNumber != messages[j].SequenceNumber {
            return messages[i].SequenceNumber < messages[j].SequenceNumber
        }
        return messages[i].ClientID.String() < messages[j].ClientID.String()
    })
    
    // Broadcast in deterministic order
    te.broadcastMessages(messages)
    
    // Advance tick
    te.currentTick.Add(1)
}

func (te *TickEngine) collectMessages() []*Message {
    messages := make([]*Message, 0)
    
    // For each connected client
    for clientID := range te.session.Clients {
        clientMessages := te.messageBuffer[clientID]
        if len(clientMessages) == 0 {
            // Send empty placeholder to maintain synchronization
            messages = append(messages, te.createEmptyMessage(clientID))
        } else {
            messages = append(messages, clientMessages...)
            te.messageBuffer[clientID] = nil
        }
    }
    
    return messages
}
```

### Message Validator

Server-side validation pipeline with rate limiting:

```go
type MessageValidator interface {
    // Validate checks if a message is valid
    Validate(msg *Message, state interface{}) error
    
    // CheckRateLimit verifies client hasn't exceeded rate limit
    CheckRateLimit(clientID uuid.UUID) error
}

type RateLimiter struct {
    limits    map[uuid.UUID]*clientRateLimit
    maxRate   int  // messages per second
    mu        sync.RWMutex
}

type clientRateLimit struct {
    count      atomic.Int32
    windowStart time.Time
    mu         sync.Mutex
}

func (rl *RateLimiter) CheckRateLimit(clientID uuid.UUID) error {
    rl.mu.RLock()
    limit := rl.limits[clientID]
    rl.mu.RUnlock()
    
    if limit == nil {
        rl.mu.Lock()
        limit = &clientRateLimit{windowStart: time.Now()}
        rl.limits[clientID] = limit
        rl.mu.Unlock()
    }
    
    limit.mu.Lock()
    defer limit.mu.Unlock()
    
    // Reset window if needed
    if time.Since(limit.windowStart) > time.Second {
        limit.count.Store(0)
        limit.windowStart = time.Now()
    }
    
    current := limit.count.Add(1)
    if int(current) > rl.maxRate {
        return fmt.Errorf("rate limit exceeded: %d messages/second", rl.maxRate)
    }
    
    return nil
}
```

### Snapshot Manager

Manages application state snapshots with delta encoding:

```go
type SnapshotManager struct {
    session          *Session
    snapshots        []*StateSnapshot
    maxSnapshots     int
    snapshotInterval time.Duration
    lastSnapshot     time.Time
    mu               sync.RWMutex
}

type StateSnapshot struct {
    ID             uuid.UUID
    SessionID      uuid.UUID
    SequenceNumber uint64  // last processed message
    Timestamp      time.Time
    Data           []byte  // serialized state
    Size           int
}

type DeltaUpdate struct {
    ID             uuid.UUID
    SessionID      uuid.UUID
    BaseSequence   uint64  // sequence number of base state
    Changes        []byte  // serialized changes
    Size           int
}

func (sm *SnapshotManager) CreateSnapshot(state interface{}, lastSeq uint64) (*StateSnapshot, error) {
    data, err := json.Marshal(state)
    if err != nil {
        return nil, err
    }
    
    snapshot := &StateSnapshot{
        ID:             uuid.New(),
        SessionID:      sm.session.ID,
        SequenceNumber: lastSeq,
        Timestamp:      time.Now(),
        Data:           data,
        Size:           len(data),
    }
    
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    // Add snapshot and maintain retention limit
    sm.snapshots = append(sm.snapshots, snapshot)
    if len(sm.snapshots) > sm.maxSnapshots {
        sm.snapshots = sm.snapshots[1:]
    }
    
    sm.lastSnapshot = time.Now()
    return snapshot, nil
}

func (sm *SnapshotManager) GenerateDelta(changes interface{}, baseSeq uint64) (*DeltaUpdate, error) {
    data, err := json.Marshal(changes)
    if err != nil {
        return nil, err
    }
    
    delta := &DeltaUpdate{
        ID:           uuid.New(),
        SessionID:    sm.session.ID,
        BaseSequence: baseSeq,
        Changes:      data,
        Size:         len(data),
    }
    
    // Check if delta is too large (>80% of snapshot)
    sm.mu.RLock()
    latestSnapshot := sm.snapshots[len(sm.snapshots)-1]
    sm.mu.RUnlock()
    
    if float64(delta.Size) > 0.8*float64(latestSnapshot.Size) {
        // Return nil to signal full snapshot should be sent instead
        return nil, fmt.Errorf("delta too large, use full snapshot")
    }
    
    return delta, nil
}

func (sm *SnapshotManager) GetLatest() *StateSnapshot {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    
    if len(sm.snapshots) == 0 {
        return nil
    }
    return sm.snapshots[len(sm.snapshots)-1]
}
```


### Reconnection Manager

Handles client disconnection and reconnection with token-based recovery:

```go
type ReconnectionManager struct {
    session      *Session
    tokens       map[string]*ReconnectionToken
    mu           sync.RWMutex
}

type ReconnectionToken struct {
    Token        string
    ClientID     uuid.UUID
    SessionID    uuid.UUID
    IssuedAt     time.Time
    ExpiresAt    time.Time
    LastSequence uint64  // last sequence number client received
}

func (rm *ReconnectionManager) IssueToken(clientID uuid.UUID) string {
    token := uuid.New().String()
    
    rm.mu.Lock()
    defer rm.mu.Unlock()
    
    rm.tokens[token] = &ReconnectionToken{
        Token:     token,
        ClientID:  clientID,
        SessionID: rm.session.ID,
        IssuedAt:  time.Now(),
        ExpiresAt: time.Now().Add(rm.session.Config.ReconnectTimeout),
    }
    
    return token
}

func (rm *ReconnectionManager) ValidateToken(token string) (*ReconnectionToken, error) {
    rm.mu.RLock()
    defer rm.mu.RUnlock()
    
    rt := rm.tokens[token]
    if rt == nil {
        return nil, fmt.Errorf("invalid reconnection token")
    }
    
    if time.Now().After(rt.ExpiresAt) {
        return nil, fmt.Errorf("reconnection token expired")
    }
    
    return rt, nil
}

func (rm *ReconnectionManager) HandleReconnect(token string, conn *tcpconn.Connection) error {
    rt, err := rm.ValidateToken(token)
    if err != nil {
        return err
    }
    
    rm.session.mu.Lock()
    client := rm.session.Clients[rt.ClientID]
    if client == nil {
        rm.session.mu.Unlock()
        return fmt.Errorf("client not found in session")
    }
    
    // Restore client connection
    client.Connection = conn
    client.Disconnected = false
    client.LastSeenTime = time.Now()
    rm.session.mu.Unlock()
    
    // Send latest snapshot
    snapshot := rm.session.SnapshotManager.GetLatest()
    if snapshot != nil {
        conn.Send(rm.encodeSnapshot(snapshot))
        
        // Send all messages since snapshot
        messages, err := rm.session.MessageStore.GetBySequenceRange(
            rm.session.ID,
            rt.ClientID,
            snapshot.SequenceNumber+1,
            ^uint64(0), // max uint64
        )
        if err == nil {
            for _, msg := range messages {
                conn.Send(rm.encodeMessage(msg))
            }
        }
    }
    
    // Send buffered messages
    client.mu.Lock()
    for _, msg := range client.BufferedMessages {
        conn.Send(rm.encodeMessage(msg))
    }
    client.BufferedMessages = nil
    client.mu.Unlock()
    
    return nil
}

func (rm *ReconnectionManager) CleanupExpiredTokens() {
    rm.mu.Lock()
    defer rm.mu.Unlock()
    
    now := time.Now()
    for token, rt := range rm.tokens {
        if now.After(rt.ExpiresAt) {
            delete(rm.tokens, token)
            
            // Remove client from session
            rm.session.mu.Lock()
            delete(rm.session.Clients, rt.ClientID)
            rm.session.mu.Unlock()
        }
    }
}
```

### Reorder Buffer (Client-Side)

Handles out-of-order message delivery on the client:

```go
type ReorderBuffer struct {
    expectedSeq uint64
    buffer      map[uint64]*Message
    holdTime    time.Duration  // 2 seconds
    timestamps  map[uint64]time.Time
    mu          sync.RWMutex
}

func (rb *ReorderBuffer) Add(msg *Message) []*Message {
    rb.mu.Lock()
    defer rb.mu.Unlock()
    
    // If this is the expected sequence, deliver it and any consecutive buffered
    if msg.SequenceNumber == rb.expectedSeq {
        delivered := []*Message{msg}
        rb.expectedSeq++
        
        // Deliver consecutive buffered messages
        for {
            next := rb.buffer[rb.expectedSeq]
            if next == nil {
                break
            }
            delivered = append(delivered, next)
            delete(rb.buffer, rb.expectedSeq)
            delete(rb.timestamps, rb.expectedSeq)
            rb.expectedSeq++
        }
        
        return delivered
    }
    
    // If sequence is in the future, buffer it
    if msg.SequenceNumber > rb.expectedSeq {
        rb.buffer[msg.SequenceNumber] = msg
        rb.timestamps[msg.SequenceNumber] = time.Now()
        return nil
    }
    
    // If sequence is in the past, it's a duplicate - discard
    return nil
}

func (rb *ReorderBuffer) CheckTimeouts() []uint64 {
    rb.mu.RLock()
    defer rb.mu.RUnlock()
    
    now := time.Now()
    timedOut := make([]uint64, 0)
    
    for seq, timestamp := range rb.timestamps {
        if now.Sub(timestamp) > rb.holdTime {
            timedOut = append(timedOut, seq)
        }
    }
    
    return timedOut
}
```

### Session Metrics

Comprehensive metrics for monitoring and debugging:

```go
type SessionMetrics struct {
    MessagesProcessed     atomic.Int64
    MessagesDelivered     atomic.Int64
    MessagesRejected      atomic.Int64
    DuplicatesDetected    atomic.Int64
    ReconnectionEvents    atomic.Int64
    SnapshotsCreated      atomic.Int64
    DeltasGenerated       atomic.Int64
    
    // Per-client metrics
    ClientMessageCounts   map[uuid.UUID]*atomic.Int64
    ClientLatencies       map[uuid.UUID]*LatencyTracker
    
    // Rejection reasons
    RejectionReasons      map[string]*atomic.Int64
    
    mu sync.RWMutex
}

type LatencyTracker struct {
    samples []time.Duration
    mu      sync.Mutex
}

func (lt *LatencyTracker) Record(latency time.Duration) {
    lt.mu.Lock()
    defer lt.mu.Unlock()
    
    lt.samples = append(lt.samples, latency)
    if len(lt.samples) > 1000 {
        lt.samples = lt.samples[1:]
    }
}

func (lt *LatencyTracker) P99() time.Duration {
    lt.mu.Lock()
    defer lt.mu.Unlock()
    
    if len(lt.samples) == 0 {
        return 0
    }
    
    sorted := make([]time.Duration, len(lt.samples))
    copy(sorted, lt.samples)
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i] < sorted[j]
    })
    
    idx := int(float64(len(sorted)) * 0.99)
    return sorted[idx]
}
```


## Data Models

### Message Lifecycle States

```
┌─────────────┐
│  Received   │
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌──────────────┐
│  Validated  │────>│   Invalid?   │
└──────┬──────┘     └──────┬───────┘
       │                   │ yes
       │ no                ▼
       │            ┌─────────────┐
       │            │  Rejected   │
       │            └─────────────┘
       ▼
┌─────────────┐     ┌──────────────┐
│  Sequenced  │────>│  Duplicate?  │
└──────┬──────┘     └──────┬───────┘
       │                   │ yes
       │ no                ▼
       │            ┌─────────────┐
       │            │  Discarded  │
       │            └─────────────┘
       ▼
┌─────────────┐
│  Persisted  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Buffered   │ (in tick engine)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Broadcast  │ (deterministic order)
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌──────────────┐
│ Pending Ack │────>│   Timeout?   │
└──────┬──────┘     └──────┬───────┘
       │                   │ yes
       │ ack received      ▼
       │            ┌─────────────┐     ┌──────────────┐
       │            │  Redeliver  │────>│ Max Attempts?│
       │            └─────────────┘     └──────┬───────┘
       │                                       │ yes
       ▼                                       ▼
┌─────────────┐                        ┌─────────────┐
│Acknowledged │                        │ Dead Letter │
└─────────────┘                        └─────────────┘
```

### Storage Schema

**Messages Table (in-memory map):**
```
messageID (uuid) → Message {
    ID               uuid.UUID
    SessionID        uuid.UUID
    ClientID         uuid.UUID
    SequenceNumber   uint64
    IdempotencyKey   string
    Payload          []byte
    Timestamp        time.Time
    DeliveryAttempts int
    Acknowledged     bool
}
```

**Idempotency Keys Table:**
```
idempotencyKey (string) → idempotencyRecord {
    key       string
    sessionID uuid.UUID
    createdAt time.Time
}
```

**Pending Acknowledgments:**
```
messageID (uuid) → pendingAck {
    messageID  uuid.UUID
    deadline   time.Time
    timerChan  <-chan time.Time
}
```

**Reconnection Tokens:**
```
token (string) → ReconnectionToken {
    Token        string
    ClientID     uuid.UUID
    SessionID    uuid.UUID
    IssuedAt     time.Time
    ExpiresAt    time.Time
    LastSequence uint64
}
```

**State Snapshots:**
```
snapshotID (uuid) → StateSnapshot {
    ID             uuid.UUID
    SessionID      uuid.UUID
    SequenceNumber uint64
    Timestamp      time.Time
    Data           []byte
    Size           int
}
```

### Protocol Extensions

The existing tcp-conn protocol is extended to support delivery guarantee features:

```go
type ControlMessage struct {
    Type      string `json:"type"`       // "join", "reconnect", "ack"
    ClientID  string `json:"client_id"`
    ChannelID string `json:"channel_id"` // used as session ID
    Token     string `json:"token,omitempty"`
}

type MessageEnvelope struct {
    Type           string `json:"type"`  // "message"
    MessageID      string `json:"message_id"`
    ClientID       string `json:"client_id"`
    SequenceNumber uint64 `json:"sequence_number"`
    IdempotencyKey string `json:"idempotency_key,omitempty"`
    Payload        []byte `json:"payload"`
    Timestamp      int64  `json:"timestamp"`
}

type AckMessage struct {
    Type      string `json:"type"`  // "ack"
    MessageID string `json:"message_id"`
}

type SnapshotMessage struct {
    Type           string `json:"type"`  // "snapshot"
    SnapshotID     string `json:"snapshot_id"`
    SequenceNumber uint64 `json:"sequence_number"`
    Data           []byte `json:"data"`
}

type DeltaMessage struct {
    Type         string `json:"type"`  // "delta"
    DeltaID      string `json:"delta_id"`
    BaseSequence uint64 `json:"base_sequence"`
    Changes      []byte `json:"changes"`
}

type ErrorMessage struct {
    Type    string `json:"type"`  // "error"
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

### Configuration Validation

Session configurations must satisfy these constraints:

```go
func (c *SessionConfig) Validate() error {
    if c.TickInterval < 50*time.Millisecond || c.TickInterval > 500*time.Millisecond {
        return errors.New("tick interval must be between 50ms and 500ms")
    }
    
    if c.AckTimeout < 100*time.Millisecond || c.AckTimeout > 5*time.Second {
        return errors.New("ack timeout must be between 100ms and 5s")
    }
    
    if c.ReconnectTimeout < 30*time.Second || c.ReconnectTimeout > 600*time.Second {
        return errors.New("reconnect timeout must be between 30s and 600s")
    }
    
    if c.SnapshotInterval < 1*time.Second {
        return errors.New("snapshot interval must be at least 1s")
    }
    
    if c.SnapshotRetention < 1 {
        return errors.New("snapshot retention must be at least 1")
    }
    
    if c.MaxBufferedMessages < 1 {
        return errors.New("max buffered messages must be at least 1")
    }
    
    if c.RateLimitPerClient < 1 {
        return errors.New("rate limit must be at least 1 message/second")
    }
    
    return nil
}
```


## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: At-Least-Once Message Delivery

*For any* message sent by a client in a session, the broker shall deliver the message to all other clients in the session one or more times (barring storage failure).

**Validates: Requirements 1.1**

### Property 2: Persistence Before Confirmation

*For any* message received from a client, if the broker confirms receipt to the client, then the message must exist in the message store.

**Validates: Requirements 1.2, 11.1**

### Property 3: Acknowledgment Lifecycle

*For any* message sent to a client, the message shall remain in the message store until an acknowledgment is received, and shall be removed within 100 milliseconds of receiving the acknowledgment. If no acknowledgment is received within 500 milliseconds, the message shall be redelivered.

**Validates: Requirements 1.3, 1.4, 1.5**

### Property 4: Unique Idempotency Key Assignment

*For any* message received from a client, the broker shall assign a unique idempotency key (either provided by the client or generated in the format "client_id:sequence_number").

**Validates: Requirements 2.1, 2.4, 2.5**

### Property 5: Duplicate Message Deduplication

*For any* message with a duplicate idempotency key, the broker shall discard the duplicate and send an acknowledgment without processing it again.

**Validates: Requirements 2.2**

### Property 6: Idempotency Key Retention

*For any* idempotency key, the broker shall track the key for at least 60 seconds after the message is acknowledged.

**Validates: Requirements 2.3**

### Property 7: Message Storage Round Trip

*For any* valid message with an idempotency key, storing the message then retrieving it shall produce a message with the same idempotency key and payload.

**Validates: Requirements 2.6**

### Property 8: Monotonic Sequence Numbers

*For any* sequence of messages from a single client, the sequence numbers shall be monotonically increasing with no gaps in the assigned sequence.

**Validates: Requirements 3.1**

### Property 9: Per-Client Sequence Isolation

*For any* two different clients in the same session, their sequence number sequences shall be independent and not affect each other.

**Validates: Requirements 3.2**

### Property 10: Sequence Number Inclusion

*For any* message delivered to clients, the message shall include the sequence number.

**Validates: Requirements 3.3**

### Property 11: Reorder Buffer Behavior

*For any* message received with a sequence number gap, the client shall buffer the message in the reorder buffer for at least 2 seconds. When a message fills the gap, all consecutive buffered messages shall be delivered in sequence order.

**Validates: Requirements 3.4, 3.5, 3.6**

### Property 12: Gap Retransmission Request

*For any* sequence number gap that persists for more than 2 seconds, the client shall request retransmission from the broker.

**Validates: Requirements 3.7**

### Property 13: Initial Snapshot Delivery

*For any* client joining a session, the broker shall send the most recent state snapshot to the client.

**Validates: Requirements 4.1**

### Property 14: Snapshot Creation Frequency

*For any* active session, the broker shall create a new state snapshot at least every 10 seconds.

**Validates: Requirements 4.2**

### Property 15: Snapshot Sequence Metadata

*For any* state snapshot created, the snapshot shall include the sequence number of the last processed message.

**Validates: Requirements 4.3**

### Property 16: Snapshot Retention Policy

*For any* session, the broker shall retain exactly the 3 most recent state snapshots, removing older snapshots as new ones are created.

**Validates: Requirements 4.4**

### Property 17: Snapshot Catch-Up Delivery

*For any* state snapshot sent to a client, the broker shall follow it with all messages that occurred after the snapshot's sequence number in order.

**Validates: Requirements 4.5**

### Property 18: Delta Contains Only Changes

*For any* delta update generated, the delta shall contain only the fields that changed since the base state.

**Validates: Requirements 5.1**

### Property 19: Delta Base Sequence Metadata

*For any* delta update, the delta shall include the sequence number of the base state it applies to.

**Validates: Requirements 5.2**

### Property 20: Delta Application Correctness

*For any* valid state and delta update, applying the delta to the base state shall produce the same result as receiving the full state.

**Validates: Requirements 5.3, 5.6**

### Property 21: Delta Size Threshold

*For any* delta update, if the delta size would exceed 80% of a full state snapshot size, the broker shall send a full snapshot instead of the delta.

**Validates: Requirements 5.5**

### Property 22: Unique Reconnection Token

*For any* client connecting to a session, the broker shall issue a unique reconnection token.

**Validates: Requirements 6.1**

### Property 23: Session State Retention

*For any* client that disconnects, the broker shall retain the session state for at least 60 seconds.

**Validates: Requirements 6.2**

### Property 24: Token-Based Reconnection

*For any* client reconnecting with a valid reconnection token, the broker shall restore the client to the session.

**Validates: Requirements 6.3**

### Property 25: Reconnection Recovery Sequence

*For any* client reconnecting, the broker shall send the most recent state snapshot followed by all messages since the snapshot in order.

**Validates: Requirements 6.4**

### Property 26: Token Expiration

*For any* reconnection token, when the token expires, the broker shall remove the associated client from the session.

**Validates: Requirements 6.5**

### Property 27: Reconnection Timeout Configuration

*For any* session configuration, the broker shall accept reconnection timeout values between 30 seconds and 600 seconds, and reject values outside this range.

**Validates: Requirements 6.6**

### Property 28: Disconnected Client Message Buffering

*For any* client that is disconnected, the broker shall buffer messages for the client up to a maximum of 1000 messages.

**Validates: Requirements 6.7**

### Property 29: Invalid Message Rejection

*For any* invalid message, the broker shall reject the message and send an error response to the client.

**Validates: Requirements 7.3**

### Property 30: Rate Limiting

*For any* client sending more than 100 messages per second, the broker shall rate-limit the client and reject excess messages.

**Validates: Requirements 7.6**

### Property 31: Rejection Logging

*For any* rejected message, the broker shall log the rejection with the client identifier, message type, and rejection reason.

**Validates: Requirements 7.7**

### Property 32: Deterministic Message Ordering

*For any* set of messages from multiple clients arriving in the same tick window, the broker shall order them deterministically by sequence number, then by client identifier for tie-breaking, and broadcast them in this order to all clients.

**Validates: Requirements 8.1, 8.2, 8.3**

### Property 33: Tick Interval Configuration

*For any* session configuration, the broker shall accept tick intervals between 50 milliseconds and 500 milliseconds, and reject values outside this range.

**Validates: Requirements 8.4**

### Property 34: Tick-Based Synchronization

*For any* tick, the broker shall broadcast all messages received during that tick in deterministic order, wait for messages from all connected clients (or send empty placeholders for missing messages), then advance to the next tick.

**Validates: Requirements 8.5, 8.6, 8.7**

### Property 35: At-Most-Once Delivery Count

*For any* message configured with at-most-once delivery, the broker shall deliver the message to clients zero or one times, never more than once.

**Validates: Requirements 10.1**

### Property 36: At-Most-Once No Persistence

*For any* message configured with at-most-once delivery, the broker shall not persist the message to the message store.

**Validates: Requirements 10.2**

### Property 37: At-Most-Once No Acknowledgment

*For any* message configured with at-most-once delivery, the broker shall not wait for acknowledgments from clients.

**Validates: Requirements 10.3**

### Property 38: At-Most-Once Failure Discard

*For any* message configured with at-most-once delivery, if a delivery failure occurs, the broker shall discard the message without retry.

**Validates: Requirements 10.4**

### Property 39: Per-Message-Type Delivery Configuration

*For any* message type, the broker shall allow configuration of delivery guarantees (at-most-once or at-least-once), and messages shall be handled according to their configured guarantee level.

**Validates: Requirements 10.6**

### Property 40: Recovery Timing

*For any* broker restart, the broker shall recover all unacknowledged messages from the message store within 5 seconds.

**Validates: Requirements 11.2**

### Property 41: Recovery Ordering Preservation

*For any* sequence of messages within a single client's sequence, the broker shall maintain the original message ordering when recovering messages after a restart.

**Validates: Requirements 11.3**

### Property 42: Atomic Message Storage

*For any* message and its sequence number, the message store shall support atomic write operations ensuring the message and sequence number are stored consistently together.

**Validates: Requirements 11.4**

### Property 43: Historical Message Retention

*For any* session that ends, the broker shall retain messages in the message store for at least 24 hours.

**Validates: Requirements 11.5**

### Property 44: Historical Message Query

*For any* session, the broker shall support querying historical messages by session identifier and sequence number range.

**Validates: Requirements 11.6**

### Property 45: Message Rate Metrics

*For any* client, the broker shall track the count of messages processed per client per second, and the tracked count shall match the actual number of messages processed.

**Validates: Requirements 12.1**

### Property 46: Latency Metrics

*For any* session, the broker shall expose metrics for message delivery latency, and the metrics shall accurately reflect actual delivery times.

**Validates: Requirements 12.2**

### Property 47: Duplicate Detection Metrics

*For any* session, the broker shall expose metrics for duplicate messages detected, and the count shall match the actual number of duplicates.

**Validates: Requirements 12.3**

### Property 48: Rejection Metrics

*For any* rejected message, the broker shall expose metrics categorized by rejection reason, and the counts shall match actual rejections.

**Validates: Requirements 12.4**

### Property 49: Size Metrics

*For any* state snapshot or delta update, the broker shall track the size, and the tracked size shall match the actual message size.

**Validates: Requirements 12.5**

### Property 50: Reconnection Metrics

*For any* reconnection event, the broker shall expose metrics for reconnection events per session, and the count shall match actual reconnections.

**Validates: Requirements 12.6**

### Property 51: Rate Limit Warning Logging

*For any* client whose message rate exceeds 100 per second, the broker shall log a warning with the client identifier.

**Validates: Requirements 12.7**


## Error Handling

### Message Validation Errors

**Invalid Message Format:**
- Reject message immediately
- Send error response to client: "invalid message format"
- Log error with message details
- Increment rejection metrics (reason: "format_error")
- Do not persist message

**Rate Limit Exceeded:**
- Reject excess messages
- Send error response: "rate limit exceeded"
- Log warning with client ID and current rate
- Increment rejection metrics (reason: "rate_limit")
- Continue accepting messages below rate limit

**Validation Failure:**
- Reject message
- Send error response with specific validation failure reason
- Log rejection with message ID and reason
- Increment rejection metrics (reason: specific validation error)
- Do not persist message

### Storage Errors

**Message Store Unavailable:**
- Reject new messages with error: "message store unavailable"
- Return error to client immediately
- Do not queue messages in memory (prevents unbounded growth)
- Log critical error
- Monitor storage health and resume when available

**Storage Write Failure:**
- Return error to client
- Do not confirm message receipt
- Log error with message ID and details
- Increment failure metrics
- Retry write with exponential backoff (max 3 attempts)

**Storage Read Failure:**
- Log error with context
- Skip affected message in recovery
- Continue processing other messages
- Expose metrics for read failures
- Alert operators if read failure rate exceeds threshold

### Network Errors

**Client Connection Failure During Delivery:**
- Mark message as unacknowledged
- Trigger redelivery according to ack timeout
- Log connection failure event
- Increment connection failure metrics
- If client has reconnection token, buffer messages

**Client Disconnection:**
- Issue reconnection token
- Start retention timer (configurable 30-600s)
- Buffer subsequent messages (max 1000)
- Log disconnection event
- Increment disconnection metrics

**Broker-to-Client Send Failure:**
- Retry send immediately (1 attempt)
- If retry fails, mark as unacknowledged
- Trigger redelivery via ack timeout mechanism
- Log send failure
- Increment send failure metrics

### Acknowledgment Errors

**Acknowledgment Timeout:**
- Increment delivery attempt count
- Redeliver message
- Log timeout event
- Increment timeout metrics
- Move to dead letter queue after max attempts (default: 5)

**Invalid Acknowledgment:**
- Log warning with message ID
- Ignore invalid ack
- Continue waiting for valid ack or timeout
- Do not modify message state

**Duplicate Acknowledgment:**
- Log debug message (message already acknowledged)
- Ignore duplicate ack
- No state change
- Do not increment metrics

**Acknowledgment for Non-Existent Message:**
- Log warning with message ID
- Send error response to client
- No state change

### Reconnection Errors

**Invalid Reconnection Token:**
- Reject reconnection attempt
- Send error response: "invalid reconnection token"
- Log rejection with token (hashed for security)
- Client must join as new participant

**Expired Reconnection Token:**
- Reject reconnection attempt
- Send error response: "reconnection token expired"
- Log expiration event
- Remove client from session
- Clean up buffered commands

**Reconnection Buffer Overflow:**
- When buffered messages exceed 1000:
  - Drop oldest messages (FIFO)
  - Log warning with client ID and dropped count
  - Continue buffering new messages
  - On reconnect, send snapshot + available buffered messages

### Tick Engine Errors

**Client Missing Message in Tick:**
- Send empty message placeholder to maintain synchronization
- Log warning with client ID and tick number
- Increment missing message metrics
- Continue tick processing

**Tick Processing Timeout:**
- If tick processing exceeds 2x tick interval:
  - Log critical error
  - Complete current tick
  - Continue with next tick
  - Alert operators

**Deterministic Ordering Failure:**
- Log critical error with message details
- Halt session to prevent state divergence
- Alert operators immediately
- Require manual intervention to resume

### Snapshot Errors

**Snapshot Generation Failure:**
- Log error with session ID
- Continue using previous snapshot
- Retry snapshot generation on next interval
- Increment snapshot failure metrics
- Alert if failures exceed threshold (3 consecutive)

**Snapshot Too Large:**
- Log warning with snapshot size
- Attempt compression
- If still too large, split into chunks
- Track large snapshot metrics

**Delta Generation Failure:**
- Fall back to full snapshot
- Log warning
- Increment delta failure metrics
- Continue normal operation

### Configuration Errors

**Invalid Configuration:**
- Reject configuration immediately
- Return descriptive error message
- Maintain previous valid configuration
- Log configuration error
- Do not start session with invalid config

**Configuration Change During Operation:**
- Reject configuration change
- Return error: "cannot change config during active session"
- Log rejection
- Configuration changes require session restart

### Recovery Errors

**Corrupted Message in Storage:**
- Log error with message ID
- Skip corrupted message
- Continue recovery with remaining messages
- Increment corruption metrics
- Alert operators

**Sequence Number Gap in Recovery:**
- Log warning with gap details
- Continue recovery
- Clients will request retransmission for gaps
- Increment gap metrics

**Recovery Timeout:**
- If recovery exceeds 5 seconds:
  - Log warning
  - Complete recovery with available messages
  - Mark session as degraded
  - Continue operation


## Testing Strategy

### Dual Testing Approach

This feature requires both unit testing and property-based testing for comprehensive coverage:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property tests**: Verify universal properties across all inputs

Both approaches are complementary and necessary. Unit tests catch concrete bugs and verify specific scenarios, while property tests verify general correctness across a wide range of inputs. Property-based tests are particularly valuable for distributed systems where message ordering and state consistency are critical.

### Property-Based Testing

**Framework:** Use [gopter](https://github.com/leanovate/gopter) for Go property-based testing.

**Configuration:**
- Minimum 100 iterations per property test (due to randomization)
- Each property test must reference its design document property
- Tag format: `// Feature: broker-delivery-guarantees, Property {number}: {property_text}`

**Property Test Coverage:**

Each of the 51 correctness properties defined in this document must be implemented as a property-based test. Key property tests include:

1. **Message Delivery Properties** (Properties 1, 2, 3):
   - Generate random messages and sessions
   - Track delivery counts and persistence timing
   - Verify at-least-once delivery and persistence-before-confirmation

2. **Idempotency Properties** (Properties 4, 5, 6, 7):
   - Generate random messages with and without idempotency keys
   - Test duplicate detection across various scenarios
   - Verify key retention and cleanup
   - Test storage round-trip with idempotency keys

3. **Sequence Number Properties** (Properties 8, 9, 10):
   - Generate random message sequences from multiple clients
   - Verify monotonic increase and per-client isolation
   - Check sequence number inclusion in messages

4. **Reorder Buffer Properties** (Properties 11, 12):
   - Generate random out-of-order message sequences
   - Verify buffering behavior and gap handling
   - Test timeout and retransmission logic

5. **Snapshot Properties** (Properties 13, 14, 15, 16, 17):
   - Generate random states and message sequences
   - Verify snapshot creation frequency and retention
   - Test catch-up delivery with snapshots + messages

6. **Delta Encoding Properties** (Properties 18, 19, 20, 21):
   - Generate random state changes
   - Verify delta correctness (applying delta = full state)
   - Test size threshold fallback to full snapshot

7. **Reconnection Properties** (Properties 22, 23, 24, 25, 26, 27, 28):
   - Generate random disconnect/reconnect scenarios
   - Verify token generation, validation, and expiration
   - Test recovery sequence and message buffering

8. **Validation Properties** (Properties 29, 30, 31):
   - Generate random valid and invalid messages
   - Test rate limiting with various message rates
   - Verify rejection logging

9. **Deterministic Ordering Properties** (Properties 32, 33, 34):
   - Generate random concurrent messages from multiple clients
   - Verify consistent ordering across all clients
   - Test tick-based synchronization

10. **At-Most-Once Properties** (Properties 35, 36, 37, 38, 39):
    - Generate random messages with different delivery guarantees
    - Verify delivery count constraints
    - Test no-persistence and no-ack behavior

11. **Recovery Properties** (Properties 40, 41, 42):
    - Simulate broker restarts with pending messages
    - Verify recovery timing and ordering preservation
    - Test atomic storage operations

12. **Historical Storage Properties** (Properties 43, 44):
    - Generate random message histories
    - Verify retention policy and query functionality

13. **Metrics Properties** (Properties 45, 46, 47, 48, 49, 50, 51):
    - Generate random operations
    - Verify metrics match actual counts and measurements
    - Test metrics categorization

### Unit Testing

Unit tests focus on specific examples, edge cases, and integration points:

**Core Functionality Tests:**
- Message delivery: single message, multiple clients
- Sequence number assignment: first message, subsequent messages
- Idempotency: duplicate detection, key generation
- Acknowledgment: timeout, redelivery, removal
- Snapshot: creation, retention, delivery
- Delta: generation, application, size threshold
- Reconnection: token issue, validation, recovery
- Tick engine: message batching, deterministic ordering
- Rate limiting: under limit, at limit, over limit

**Edge Cases:**
- Empty message payload
- Very large message payload (test limits)
- Zero clients in session
- Single client in session
- Maximum clients (10 clients)
- Concurrent acknowledgments for same message
- Acknowledgment for non-existent message
- Reconnection with expired token
- Reconnection buffer overflow (>1000 messages)
- Snapshot generation failure
- Delta larger than snapshot
- Sequence number wraparound (uint64 max)
- Tick with no messages from any client
- Tick with messages from all clients
- Storage failure during persistence
- Storage recovery after failure
- Broker restart with pending messages
- Configuration validation: valid and invalid values

**Integration Tests:**
- Multiple sessions with different configurations
- Multiple clients per session
- Message flow: receive → validate → sequence → persist → broadcast → ack
- Reconnection flow: disconnect → buffer → reconnect → recover
- Snapshot flow: create → retain → deliver → catch-up
- Tick flow: collect → order → broadcast → advance
- Cross-component: sequence manager + message store
- Cross-component: tick engine + deterministic ordering
- Cross-component: reconnection manager + snapshot manager

**Error Handling Tests:**
- Storage unavailable scenarios
- Network failures during delivery
- Invalid message formats
- Rate limit exceeded
- Validation failures
- Acknowledgment timeout
- Reconnection token expiration
- Configuration errors
- Recovery errors

### Benchmark Testing

Performance requirements (Requirements 9.1-9.7) must be verified with benchmarks:

```go
BenchmarkMessageDelivery-12           // Target: <50ms p99 latency
BenchmarkMessageThroughput-12         // Target: >300 msg/s (10 clients)
BenchmarkBroadcast10Clients-12        // Target: <5ms
BenchmarkMessageValidation-12         // Target: <1ms per message
BenchmarkSnapshotGeneration-12        // Target: <100ms for 10MB state
BenchmarkDeltaGeneration-12           // Target: <10ms
BenchmarkAtMostOnceLatency-12         // Target: <10ms p99
BenchmarkSequenceAssignment-12        // Measure overhead
BenchmarkIdempotencyCheck-12          // Measure overhead
BenchmarkReorderBuffer-12             // Measure overhead
BenchmarkTickProcessing-12            // Measure tick overhead
```

Benchmarks should measure:
- End-to-end latency (message received to delivered)
- Throughput (messages per second)
- Memory allocations per operation
- Storage operation overhead
- CPU usage under load
- Latency percentiles (p50, p95, p99)

### Test Data Generation

For property-based tests, generators should produce:

**Message Generators:**
- Random payloads (various sizes: 0 bytes, 1KB, 10KB, 100KB)
- Random message IDs (valid UUIDs)
- Random client IDs (valid UUIDs)
- Random session IDs (valid UUIDs)
- Random idempotency keys (strings, UUIDs, empty)
- Random timestamps
- Random sequence numbers

**Configuration Generators:**
- Random tick intervals (valid: 50-500ms, invalid: outside range)
- Random ack timeouts (valid: 100ms-5s, invalid: outside range)
- Random reconnect timeouts (valid: 30-600s, invalid: outside range)
- Random snapshot intervals (valid: ≥1s, invalid: <1s)
- Random retention counts (valid: ≥1, invalid: <1)
- Random rate limits (valid: ≥1, invalid: <1)

**Scenario Generators:**
- Random delivery success/failure patterns
- Random acknowledgment timing (before/after timeout)
- Random connection failure points
- Random storage availability patterns
- Random message arrival patterns (bursty, steady, sparse)
- Random client disconnect/reconnect patterns
- Random out-of-order sequence patterns

**State Generators:**
- Random states (various sizes: 1KB, 100KB, 1MB, 10MB)
- Random state changes (small deltas, large deltas)
- Random entity counts (1-1000 entities)

### Test Isolation

Each test must be isolated:
- Create fresh broker instance per test
- Use separate in-memory storage per test
- Clean up goroutines and timers after test
- No shared state between tests
- Use unique session IDs per test
- Use unique client IDs per test
- Close all connections after test

### Continuous Testing

- Run unit tests on every commit
- Run property tests on every commit (100 iterations minimum)
- Run extended property tests nightly (1000 iterations)
- Run benchmarks on release candidates
- Track performance regression over time
- Alert on property test failures (indicates correctness bug)
- Alert on benchmark regression (>10% latency increase)

### Test Coverage Goals

- Line coverage: >85%
- Branch coverage: >80%
- Property coverage: 100% (all 51 properties tested)
- Error path coverage: >90%
- Integration test coverage: all major flows


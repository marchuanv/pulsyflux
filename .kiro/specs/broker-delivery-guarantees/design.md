# Design Document: Broker Delivery Guarantees

## Overview

This design extends the existing PulsyFlux broker to support configurable delivery guarantees: at-most-once, at-least-once, and exactly-once. The current broker provides a lightweight pub/sub system with no persistence or acknowledgments. This enhancement adds message durability, acknowledgment tracking, idempotency, and configurable reliability levels while maintaining the broker's performance characteristics.

The design introduces three core subsystems:
1. **Message Store**: Persistent storage for messages and metadata
2. **Acknowledgment Manager**: Tracks message delivery and handles timeouts
3. **Idempotency Manager**: Deduplicates messages for exactly-once delivery

Each channel can be configured with a specific delivery guarantee level, allowing applications to optimize for their reliability and performance requirements.

### Design Goals

- Support three delivery guarantee levels per channel
- Maintain backward compatibility with existing broker API
- Minimize performance impact for at-most-once delivery (current behavior)
- Provide durable message storage for at-least-once and exactly-once modes
- Enable automatic message redelivery with configurable timeouts
- Implement efficient deduplication for exactly-once delivery
- Expose metrics for monitoring delivery guarantees

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Broker Server                            │
│                                                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │   Channel    │    │   Channel    │    │   Channel    │      │
│  │  (at-most-   │    │ (at-least-   │    │  (exactly-   │      │
│  │    once)     │    │    once)     │    │    once)     │      │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘      │
│         │                   │                   │               │
│         └───────────────────┼───────────────────┘               │
│                             │                                   │
│         ┌───────────────────┴───────────────────┐               │
│         │                                       │               │
│  ┌──────▼──────┐    ┌──────────────┐    ┌──────▼──────┐       │
│  │   Message   │    │Acknowledgment│    │ Idempotency │       │
│  │    Store    │    │   Manager    │    │   Manager   │       │
│  └─────────────┘    └──────────────┘    └─────────────┘       │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```


### Component Interactions

**At-Most-Once Flow:**
```
Producer → Broker → Consumer
           (no persistence, no ack)
```

**At-Least-Once Flow:**
```
Producer → Broker → Message Store → Consumer
           ↓                         ↓
           Confirm                   Ack
           ↓                         ↓
           Ack Manager ← ─ ─ ─ ─ ─ ─ ┘
           (retry on timeout)
```

**Exactly-Once Flow:**
```
Producer → Broker → Idempotency Check → Message Store → Consumer
           ↓        (deduplicate)        ↓               ↓
           Confirm                       Ack Manager     Ack
                                         (retry)         ↓
                                         ← ─ ─ ─ ─ ─ ─ ─ ┘
```

### Delivery Guarantee Modes

**At-Most-Once:**
- No persistence
- No acknowledgments
- Immediate message removal after send
- Lowest latency, highest throughput
- Messages may be lost on failure

**At-Least-Once:**
- Messages persisted before delivery
- Acknowledgment required
- Retry on timeout
- Messages may be duplicated
- No message loss (barring storage failure)

**Exactly-Once:**
- All at-least-once features
- Idempotency key tracking
- Deduplication on producer and consumer side
- Highest reliability, highest latency
- No loss, no duplication


## Components and Interfaces

### Channel Configuration

Each channel maintains a configuration that specifies its delivery guarantee level:

```go
type DeliveryGuarantee int

const (
    AtMostOnce DeliveryGuarantee = iota
    AtLeastOnce
    ExactlyOnce
)

type ChannelConfig struct {
    ID                   uuid.UUID
    DeliveryGuarantee    DeliveryGuarantee
    AckTimeout           time.Duration  // 1s - 300s
    MaxDeliveryAttempts  int            // default: 5
    RetryBackoffBase     time.Duration  // 100ms - 60s
    IdempotencyRetention time.Duration  // 1h - 168h (exactly-once only)
}
```

### Message Store Interface

The message store provides persistent storage for messages and metadata:

```go
type MessageStore interface {
    // Store persists a message atomically with its metadata
    Store(msg *StoredMessage) error
    
    // Get retrieves a message by ID
    Get(messageID uuid.UUID) (*StoredMessage, error)
    
    // Delete removes a message from storage
    Delete(messageID uuid.UUID) error
    
    // GetUnacknowledged returns messages awaiting acknowledgment for a channel
    GetUnacknowledged(channelID uuid.UUID) ([]*StoredMessage, error)
    
    // UpdateDeliveryAttempt increments delivery attempt count and timestamp
    UpdateDeliveryAttempt(messageID uuid.UUID) error
    
    // Close cleanly shuts down the store
    Close() error
}

type StoredMessage struct {
    ID               uuid.UUID
    ChannelID        uuid.UUID
    Payload          []byte
    IdempotencyKey   string
    DeliveryAttempts int
    LastAttemptTime  time.Time
    CreatedAt        time.Time
    Acknowledged     bool
}
```

**Implementation:** The initial implementation will use an in-memory map with mutex protection for simplicity. Future versions can add disk-based persistence (e.g., BadgerDB, BoltDB) for true durability across restarts.


### Acknowledgment Manager

The acknowledgment manager tracks message delivery and handles timeouts:

```go
type AckManager interface {
    // TrackDelivery registers a message as pending acknowledgment
    TrackDelivery(messageID uuid.UUID, channelID uuid.UUID, timeout time.Duration) error
    
    // Acknowledge marks a message as successfully delivered
    Acknowledge(messageID uuid.UUID) error
    
    // NegativeAcknowledge marks a message for immediate redelivery
    NegativeAcknowledge(messageID uuid.UUID) error
    
    // GetTimedOut returns messages that have exceeded their ack timeout
    GetTimedOut() []uuid.UUID
    
    // Stop shuts down the manager
    Stop()
}

type pendingAck struct {
    messageID  uuid.UUID
    channelID  uuid.UUID
    deadline   time.Time
    timerChan  <-chan time.Time
}
```

**Implementation:** Uses a map of pending acknowledgments with timer goroutines. A background worker periodically checks for timed-out messages and triggers redelivery.

### Idempotency Manager

The idempotency manager handles deduplication for exactly-once delivery:

```go
type IdempotencyManager interface {
    // CheckAndStore checks if a key exists; if not, stores it atomically
    // Returns true if this is a new key (not a duplicate)
    CheckAndStore(key string, channelID uuid.UUID) (bool, error)
    
    // Remove deletes an idempotency key (after retention period)
    Remove(key string) error
    
    // Cleanup removes expired keys based on retention policy
    Cleanup(retention time.Duration) error
}

type idempotencyRecord struct {
    key       string
    channelID uuid.UUID
    createdAt time.Time
}
```

**Implementation:** Uses an in-memory map with periodic cleanup. Keys are stored with timestamps and removed after the configured retention period.


### Enhanced Channel Structure

The existing channel structure is extended to support delivery guarantees:

```go
type channel struct {
    id               uuid.UUID
    config           *ChannelConfig
    clients          map[uuid.UUID]*tcpconn.Connection
    messageStore     MessageStore
    ackManager       AckManager
    idempotencyMgr   IdempotencyManager
    metrics          *ChannelMetrics
    mu               sync.RWMutex
    redeliveryWorker *RedeliveryWorker
}

type ChannelMetrics struct {
    messagesPublished    atomic.Int64
    messagesDelivered    atomic.Int64
    messagesFailed       atomic.Int64
    duplicatesDetected   atomic.Int64
    deliveryAttempts     atomic.Int64
}
```

### Redelivery Worker

A background worker handles message redelivery for timed-out acknowledgments:

```go
type RedeliveryWorker struct {
    channel      *channel
    checkInterval time.Duration
    stopChan     chan struct{}
}

func (w *RedeliveryWorker) Start() {
    ticker := time.NewTicker(w.checkInterval)
    go func() {
        for {
            select {
            case <-ticker.C:
                w.processTimedOut()
            case <-w.stopChan:
                ticker.Stop()
                return
            }
        }
    }()
}

func (w *RedeliveryWorker) processTimedOut() {
    timedOut := w.channel.ackManager.GetTimedOut()
    for _, msgID := range timedOut {
        msg, err := w.channel.messageStore.Get(msgID)
        if err != nil {
            continue
        }
        
        if msg.DeliveryAttempts >= w.channel.config.MaxDeliveryAttempts {
            w.moveToDeadLetter(msg)
            continue
        }
        
        w.channel.messageStore.UpdateDeliveryAttempt(msgID)
        w.redeliver(msg)
    }
}
```


### Protocol Extensions

The existing protocol is extended to support acknowledgments:

```go
type MessageEnvelope struct {
    MessageID      string `json:"message_id"`
    IdempotencyKey string `json:"idempotency_key,omitempty"`
    Payload        []byte `json:"payload"`
}

type AckMessage struct {
    MessageID string `json:"message_id"`
    Success   bool   `json:"success"`  // false = negative ack
}
```

**Wire Protocol:**
- Control messages continue to use the existing JSON format
- Message envelopes wrap payloads with metadata
- Acknowledgments are sent on a dedicated logical connection per client
- Ack connection uses a reserved UUID: `ffffffff-ffff-ffff-ffff-ffffffffffff`

### API Extensions

**Server API:**
```go
// NewServerWithConfig creates a server with default channel configs
func NewServerWithConfig(address string, defaultConfig *ChannelConfig) *Server

// SetChannelConfig configures delivery guarantees for a specific channel
func (s *Server) SetChannelConfig(channelID uuid.UUID, config *ChannelConfig) error

// GetMetrics returns delivery metrics for a channel
func (s *Server) GetMetrics(channelID uuid.UUID) (*ChannelMetrics, error)
```

**Client API:**
```go
// PublishWithIdempotency publishes a message with a custom idempotency key
func (c *Client) PublishWithIdempotency(payload []byte, idempotencyKey string) error

// EnableAcknowledgments starts the acknowledgment handler
func (c *Client) EnableAcknowledgments() error

// Acknowledge sends a positive acknowledgment for a message
func (c *Client) Acknowledge(messageID string) error

// NegativeAcknowledge requests immediate redelivery
func (c *Client) NegativeAcknowledge(messageID string) error
```


## Data Models

### Message Lifecycle States

```
┌─────────────┐
│  Published  │
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌──────────────┐
│  Persisted  │────>│  Duplicate?  │ (exactly-once only)
└──────┬──────┘     └──────┬───────┘
       │                   │ yes
       │ no                ▼
       │            ┌─────────────┐
       │            │  Discarded  │
       │            └─────────────┘
       ▼
┌─────────────┐
│  Delivered  │
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
messageID (uuid) → StoredMessage {
    ID               uuid.UUID
    ChannelID        uuid.UUID
    Payload          []byte
    IdempotencyKey   string
    DeliveryAttempts int
    LastAttemptTime  time.Time
    CreatedAt        time.Time
    Acknowledged     bool
}
```

**Idempotency Keys Table:**
```
idempotencyKey (string) → idempotencyRecord {
    key       string
    channelID uuid.UUID
    createdAt time.Time
}
```

**Pending Acknowledgments:**
```
messageID (uuid) → pendingAck {
    messageID  uuid.UUID
    channelID  uuid.UUID
    deadline   time.Time
    timerChan  <-chan time.Time
}
```


### Configuration Validation

Channel configurations must satisfy these constraints:

```go
func (c *ChannelConfig) Validate() error {
    if c.AckTimeout < 1*time.Second || c.AckTimeout > 300*time.Second {
        return errors.New("ack timeout must be between 1s and 300s")
    }
    
    if c.RetryBackoffBase < 100*time.Millisecond || c.RetryBackoffBase > 60*time.Second {
        return errors.New("retry backoff base must be between 100ms and 60s")
    }
    
    if c.DeliveryGuarantee == ExactlyOnce {
        if c.IdempotencyRetention < 1*time.Hour || c.IdempotencyRetention > 168*time.Hour {
            return errors.New("idempotency retention must be between 1h and 168h")
        }
    }
    
    if c.MaxDeliveryAttempts < 1 {
        return errors.New("max delivery attempts must be at least 1")
    }
    
    return nil
}
```

### Dead Letter Queue

Messages that exceed max delivery attempts are moved to a dead letter queue:

```go
type DeadLetterQueue struct {
    messages map[uuid.UUID]*DeadLetterMessage
    mu       sync.RWMutex
}

type DeadLetterMessage struct {
    OriginalMessage  *StoredMessage
    FailureReason    string
    FailedAt         time.Time
    TotalAttempts    int
}
```


## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: At-Most-Once Delivery Count

For any message in a channel configured with at-most-once delivery guarantee, the message shall be delivered to consumers zero or one times, never more than once.

**Validates: Requirements 1.1**

### Property 2: At-Most-Once No Persistence

For any message in a channel configured with at-most-once delivery guarantee, the message shall not be persisted to the message store at any point during its lifecycle.

**Validates: Requirements 1.2, 6.4**

### Property 3: At-Most-Once No Acknowledgment Wait

For any message in a channel configured with at-most-once delivery guarantee, the publish operation shall complete without waiting for consumer acknowledgments.

**Validates: Requirements 1.3**

### Property 4: At-Most-Once Failure Discard

For any message in a channel configured with at-most-once delivery guarantee, if a delivery failure occurs, the message shall be discarded and not retried.

**Validates: Requirements 1.4**

### Property 5: At-Least-Once Delivery Count

For any message in a channel configured with at-least-once delivery guarantee, the message shall be delivered to consumers one or more times, never zero times (barring storage failure).

**Validates: Requirements 2.1**

### Property 6: At-Least-Once Persistence Until Ack

For any message in a channel configured with at-least-once delivery guarantee, the message shall remain in the message store from the time it is sent until an acknowledgment is received.

**Validates: Requirements 2.2**


### Property 7: Redelivery On Timeout

For any message in a channel configured with at-least-once or exactly-once delivery guarantee, if the acknowledgment timeout expires without receiving an acknowledgment, the message shall be redelivered with the same idempotency key (if applicable).

**Validates: Requirements 2.3, 3.5**

### Property 8: Acknowledgment Removal Timing

For any message in a channel configured with at-least-once or exactly-once delivery guarantee, when an acknowledgment is received, the message shall be removed from the message store within 1 second.

**Validates: Requirements 2.4**

### Property 9: Persistence Before Producer Confirmation

For any message in a channel configured with at-least-once or exactly-once delivery guarantee, the message shall be persisted to the message store before the broker confirms receipt to the producer.

**Validates: Requirements 2.5, 3.6, 6.1**

### Property 10: Exactly-Once Delivery Count

For any message in a channel configured with exactly-once delivery guarantee, the message shall be delivered to consumers exactly one time, never zero and never more than one.

**Validates: Requirements 3.1**

### Property 11: Idempotency Key Uniqueness

For any message in a channel configured with exactly-once delivery guarantee, the message shall have a unique idempotency key, either provided by the producer or generated by the broker.

**Validates: Requirements 3.2, 7.1, 7.2, 7.3**

### Property 12: Duplicate Message Discard

For any message in a channel configured with exactly-once delivery guarantee, if a message with a duplicate idempotency key is received, the duplicate shall be discarded and not delivered.

**Validates: Requirements 3.3**

### Property 13: Idempotency Key Retention

For any idempotency key in a channel configured with exactly-once delivery guarantee, the key shall be tracked for the configured retention period and removed after the retention period expires.

**Validates: Requirements 3.4, 7.5**


### Property 14: Per-Channel Configuration

For any channel, the broker shall allow configuration of a delivery guarantee level, and messages sent to that channel shall be handled according to the configured level.

**Validates: Requirements 4.1, 4.2**

### Property 15: Configuration Validation

For any channel configuration, the broker shall validate that all configuration parameters are within their valid ranges (delivery guarantee level is one of the three valid values, ack timeout is 1-300s, retry backoff is 100ms-60s, idempotency retention is 1h-168h), and shall reject invalid configurations with descriptive error messages.

**Validates: Requirements 4.3, 4.4, 5.2**

### Property 16: Acknowledgment Pending State

For any message in a channel that requires acknowledgments (at-least-once or exactly-once), after delivery the message shall remain in a pending state until an acknowledgment is received.

**Validates: Requirements 5.1**

### Property 17: Acknowledgment Marks Delivery Complete

For any message in a channel that requires acknowledgments, when an acknowledgment is received, the message shall be marked as successfully delivered.

**Validates: Requirements 5.3**

### Property 18: Negative Acknowledgment Triggers Redelivery

For any message in a channel that requires acknowledgments, if a negative acknowledgment is received, the message shall be redelivered according to the configured delivery guarantee level.

**Validates: Requirements 5.4**

### Property 19: Acknowledgment Exclusivity

For any message in a channel that requires acknowledgments, while waiting for an acknowledgment, the message shall not be delivered to another consumer in the same consumer group.

**Validates: Requirements 5.5**

### Property 20: Recovery Of Unacknowledged Messages

For any unacknowledged message in the message store, when the broker restarts, the message shall be recovered and made available for redelivery.

**Validates: Requirements 6.2**


### Property 21: Message Ordering During Recovery

For any sequence of messages within a single channel, the broker shall maintain the original message ordering when recovering messages after a restart.

**Validates: Requirements 6.3**

### Property 22: Message Storage Round Trip

For any valid message with an idempotency key, storing the message with its idempotency key then retrieving it shall produce a message with the same idempotency key and payload.

**Validates: Requirements 7.6**

### Property 23: Idempotency Key Storage With Retention

For any idempotency key in exactly-once mode, the key shall be stored in the message store with the configured retention period between 1 hour and 168 hours.

**Validates: Requirements 7.4**

### Property 24: Delivery Attempt Tracking

For any message that is delivered, the broker shall track and increment the delivery attempt count and record the timestamp of each delivery attempt.

**Validates: Requirements 8.1, 8.6**

### Property 25: Delivery Metrics Exposure

For any delivery guarantee level, the broker shall expose metrics for successful deliveries, failed deliveries, and (for exactly-once mode) duplicate messages detected.

**Validates: Requirements 8.2, 8.3, 8.4**

### Property 26: Dead Letter Queue On Max Attempts

For any message that exceeds the configured maximum delivery attempts threshold, the message shall be moved to a dead letter queue.

**Validates: Requirements 8.5**

### Property 27: Storage Unavailable Rejection

For any message, if the message store is unavailable, the broker shall reject the message with a descriptive error.

**Validates: Requirements 9.1**

### Property 28: Storage Recovery Timing

For any broker instance, when the message store becomes available after being unavailable, the broker shall resume accepting messages within 5 seconds.

**Validates: Requirements 9.2**


### Property 29: Connection Failure Redelivery

For any message in a channel that requires acknowledgments, if a consumer connection fails during delivery, the message shall be redelivered according to the configured delivery guarantee level.

**Validates: Requirements 9.3**

### Property 30: Exponential Backoff For Redelivery

For any message that is redelivered multiple times, the delay between redelivery attempts shall increase exponentially based on the configured base delay (between 100ms and 60s).

**Validates: Requirements 9.4**

### Property 31: Max Attempts Failure Logging

For any message that cannot be delivered after the maximum delivery attempts, the broker shall log the failure with the message identifier and error details.

**Validates: Requirements 9.5**


## Error Handling

### Storage Errors

**Message Store Unavailable:**
- Reject new messages with error: "message store unavailable"
- Return error to producer immediately
- Do not queue messages in memory (prevents unbounded memory growth)
- Monitor storage health and resume when available

**Storage Write Failure:**
- Return error to producer
- Do not confirm message receipt
- Log error with message ID and details
- Increment failure metrics

**Storage Read Failure:**
- Log error with context
- Skip affected message in recovery
- Continue processing other messages
- Expose metrics for read failures

### Acknowledgment Errors

**Acknowledgment Timeout:**
- Increment delivery attempt count
- Apply exponential backoff
- Redeliver message
- Move to dead letter queue after max attempts

**Invalid Acknowledgment:**
- Log warning with message ID
- Ignore invalid ack
- Continue waiting for valid ack or timeout

**Duplicate Acknowledgment:**
- Log warning (message already acknowledged)
- Ignore duplicate ack
- No state change

### Configuration Errors

**Invalid Configuration:**
- Reject configuration immediately
- Return descriptive error message
- Maintain previous valid configuration
- Log configuration error

**Configuration Change During Operation:**
- Apply new configuration to new messages only
- Complete in-flight messages with old configuration
- Log configuration change event


### Network Errors

**Consumer Connection Failure:**
- Detect via tcp-conn error
- Mark message as unacknowledged
- Trigger redelivery according to delivery guarantee level
- Log connection failure event

**Producer Connection Failure:**
- Return error to producer (if still connected)
- Do not persist message if not yet stored
- Clean up partial state
- Log connection failure

### Idempotency Errors

**Duplicate Message Detected:**
- Discard duplicate silently
- Increment duplicate detection metric
- Return success to producer (idempotent operation)
- Log duplicate detection at debug level

**Idempotency Key Collision:**
- Treat as duplicate (same key = same message)
- Follow duplicate message handling
- Do not deliver duplicate

### Dead Letter Queue

**Message Moved to DLQ:**
- Log with full message details
- Include failure reason and attempt count
- Expose DLQ metrics
- Provide API to inspect/replay DLQ messages

**DLQ Full:**
- Log critical error
- Continue moving messages to DLQ (no limit)
- Alert operators via metrics
- Consider implementing DLQ size limits in future


## Testing Strategy

### Dual Testing Approach

This feature requires both unit testing and property-based testing for comprehensive coverage:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property tests**: Verify universal properties across all inputs

Both approaches are complementary and necessary. Unit tests catch concrete bugs and verify specific scenarios, while property tests verify general correctness across a wide range of inputs.

### Property-Based Testing

**Framework:** Use [gopter](https://github.com/leanovate/gopter) for Go property-based testing.

**Configuration:**
- Minimum 100 iterations per property test (due to randomization)
- Each property test must reference its design document property
- Tag format: `// Feature: broker-delivery-guarantees, Property {number}: {property_text}`

**Property Test Coverage:**

Each of the 31 correctness properties defined in this document must be implemented as a property-based test. Key property tests include:

1. **Delivery Count Properties** (Properties 1, 5, 10):
   - Generate random messages and channels with different delivery guarantees
   - Track actual delivery counts
   - Verify counts match guarantee level constraints

2. **Persistence Properties** (Properties 2, 6, 9):
   - Generate random messages and delivery guarantee configurations
   - Verify persistence behavior matches configuration
   - Check timing of persistence relative to confirmations

3. **Redelivery Properties** (Properties 7, 18, 29, 30):
   - Generate random timeout scenarios
   - Verify redelivery occurs with correct timing
   - Check exponential backoff calculations

4. **Idempotency Properties** (Properties 11, 12, 13, 22, 23):
   - Generate random messages with and without idempotency keys
   - Test duplicate detection across various scenarios
   - Verify key retention and cleanup

5. **Configuration Properties** (Properties 14, 15):
   - Generate random valid and invalid configurations
   - Verify validation logic
   - Test configuration application per channel


### Unit Testing

Unit tests focus on specific examples, edge cases, and integration points:

**Core Functionality Tests:**
- At-most-once: single message delivery, no persistence
- At-least-once: message persistence, acknowledgment, redelivery
- Exactly-once: idempotency key generation, duplicate detection
- Configuration: valid/invalid config handling
- Dead letter queue: max attempts exceeded

**Edge Cases:**
- Empty message payload
- Very large message payload (test limits)
- Concurrent acknowledgments for same message
- Acknowledgment for non-existent message
- Storage failure during persistence
- Storage recovery after failure
- Broker restart with pending messages
- Idempotency key retention expiration
- Zero timeout (invalid)
- Negative timeout (invalid)

**Integration Tests:**
- Multiple channels with different delivery guarantees
- Multiple producers and consumers
- Message ordering within channel
- Cross-channel isolation
- Metrics accuracy across operations

**Error Handling Tests:**
- Storage unavailable scenarios
- Network failures during delivery
- Invalid acknowledgment messages
- Configuration errors
- Timeout edge cases

### Benchmark Testing

Performance requirements (Requirements 10.1-10.6) must be verified with benchmarks:

```go
BenchmarkAtMostOnce-12      // Target: <10ms p99 latency
BenchmarkAtLeastOnce-12     // Target: <50ms p99 latency
BenchmarkExactlyOnce-12     // Target: <100ms p99 latency
BenchmarkThroughputAtMostOnce-12    // Target: >10,000 msg/s
BenchmarkThroughputAtLeastOnce-12   // Target: >5,000 msg/s
BenchmarkThroughputExactlyOnce-12   // Target: >2,000 msg/s
```

Benchmarks should measure:
- End-to-end latency (publish to delivery)
- Throughput (messages per second)
- Memory allocations per operation
- Storage operation overhead


### Test Data Generation

For property-based tests, generators should produce:

**Message Generators:**
- Random payloads (various sizes: 0 bytes, 1KB, 10KB, 100KB)
- Random message IDs (valid UUIDs)
- Random idempotency keys (strings, UUIDs, empty)
- Random timestamps

**Configuration Generators:**
- Random delivery guarantee levels (all three types)
- Random timeout values (valid and invalid ranges)
- Random retry backoff values (valid and invalid ranges)
- Random retention periods (valid and invalid ranges)
- Random max attempt counts (1-100)

**Scenario Generators:**
- Random delivery success/failure patterns
- Random acknowledgment timing (before/after timeout)
- Random connection failure points
- Random storage availability patterns

### Test Isolation

Each test must be isolated:
- Create fresh broker instance per test
- Use separate in-memory storage per test
- Clean up goroutines and timers after test
- No shared state between tests
- Use unique channel IDs per test

### Continuous Testing

- Run unit tests on every commit
- Run property tests on every commit (100 iterations minimum)
- Run benchmarks on release candidates
- Track performance regression over time
- Alert on property test failures (indicates correctness bug)


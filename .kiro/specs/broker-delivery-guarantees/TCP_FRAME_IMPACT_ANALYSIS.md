# TCP Frame Impact Analysis

## Question
Will the delivery guarantees spec changes affect the tcp-conn frame format?

## Answer: NO - The tcp-conn frame format remains unchanged

The spec operates at the **application layer** (broker protocol), not the **transport layer** (tcp-conn framing).

---

## Current TCP-Conn Frame Format

The existing tcp-conn uses this frame structure:

```
┌──────────┬────────────┬─────────────┬─────────────┬──────────┐
│ ID Length│     ID     │ Total Length│ Chunk Length│   Data   │
│  (1 byte)│ (N bytes)  │  (4 bytes)  │  (4 bytes)  │(M bytes) │
└──────────┴────────────┴─────────────┴─────────────┴──────────┘
```

**Fields:**
- `ID Length` (1 byte): Length of the connection ID string
- `ID` (N bytes): Connection UUID as string
- `Total Length` (4 bytes): Total message size (for chunking)
- `Chunk Length` (4 bytes): Current chunk size
- `Data` (M bytes): Actual payload

**Purpose:** Multiplexing multiple logical connections over a single TCP socket.

**Code location:** `pulsyflux/tcp-conn/logical.go` (Send method, lines ~200-230)

---

## Spec Protocol Extensions

The spec adds **application-level message types** that are sent **inside** the tcp-conn payload:

```go
// These are JSON messages sent as tcp-conn payloads
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
```

**These are broker-level messages, not tcp-conn frames.**

---

## Layer Separation

```
┌─────────────────────────────────────────────────────────┐
│                   Application Layer                      │
│  (Broker Protocol - NEW)                                 │
│  - MessageEnvelope with sequence numbers                 │
│  - AckMessage for acknowledgments                        │
│  - SnapshotMessage for state sync                        │
│  - JSON-encoded messages                                 │
└────────────────────┬────────────────────────────────────┘
                     │
                     │ (sent as payload)
                     ▼
┌─────────────────────────────────────────────────────────┐
│                   Transport Layer                        │
│  (TCP-Conn Framing - UNCHANGED)                          │
│  - ID Length + ID + Total Length + Chunk Length + Data   │
│  - Multiplexing by connection UUID                       │
│  - Chunking for large messages                           │
└─────────────────────────────────────────────────────────┘
```

---

## How It Works

### Before (Current Broker):

1. Client calls `broker.Publish(payload)`
2. Broker calls `tcpconn.Send(payload)` directly
3. TCP-conn wraps payload in frame: `[ID_LEN][ID][TOTAL_LEN][CHUNK_LEN][payload]`
4. Frame sent over TCP socket

### After (With Delivery Guarantees):

1. Client calls `broker.Publish(payload)`
2. Broker creates `MessageEnvelope`:
   ```json
   {
     "type": "message",
     "message_id": "uuid",
     "sequence_number": 42,
     "payload": <original payload>
   }
   ```
3. Broker serializes envelope to JSON
4. Broker calls `tcpconn.Send(json_bytes)`
5. TCP-conn wraps JSON in frame: `[ID_LEN][ID][TOTAL_LEN][CHUNK_LEN][json_bytes]`
6. Frame sent over TCP socket

**The tcp-conn frame format is identical. Only the payload content changes.**

---

## What Changes

### At the Broker Level (Application Layer):
- ✅ Messages now wrapped in JSON envelopes
- ✅ Sequence numbers added to messages
- ✅ Acknowledgment messages sent
- ✅ Snapshot messages sent
- ✅ Message store persists envelopes

### At the TCP-Conn Level (Transport Layer):
- ❌ Frame format unchanged
- ❌ Multiplexing unchanged
- ❌ Chunking unchanged
- ❌ Connection pooling unchanged

---

## Backward Compatibility

### TCP-Conn Layer:
- ✅ **Fully backward compatible** - frame format unchanged
- ✅ Existing tcp-conn clients can still connect
- ✅ No changes to connection pooling or multiplexing

### Broker Layer:
- ⚠️ **Breaking change** - message format changes from raw bytes to JSON envelopes
- ❌ Old broker clients won't understand new message format
- ❌ New broker clients won't understand old message format
- ✅ Can be mitigated with versioning or feature flags

---

## Migration Path

If you want to maintain backward compatibility at the broker level:

### Option 1: Version Field
Add a version field to distinguish old vs new protocol:

```go
type MessageEnvelope struct {
    Version        int    `json:"version"`  // 1 = old, 2 = new
    Type           string `json:"type,omitempty"`
    MessageID      string `json:"message_id,omitempty"`
    SequenceNumber uint64 `json:"sequence_number,omitempty"`
    Payload        []byte `json:"payload"`
}
```

### Option 2: Feature Negotiation
During connection handshake, negotiate which features to use:

```go
type ControlMessage struct {
    Type     string   `json:"type"`  // "join"
    Features []string `json:"features"`  // ["delivery-guarantees", "snapshots"]
}
```

### Option 3: Separate Channels
Use different channel IDs for old vs new protocol:
- Old clients: Use existing channels (raw payload)
- New clients: Use new channels (JSON envelopes)

---

## Performance Impact

### TCP-Conn Layer:
- ✅ **No impact** - frame format unchanged
- ✅ Same overhead per message
- ✅ Same throughput characteristics

### Broker Layer:
- ⚠️ **JSON serialization overhead** - adds ~100-500 bytes per message
- ⚠️ **Parsing overhead** - JSON encode/decode adds ~1-5µs per message
- ✅ **Acceptable for target workload** - 300 msg/s is well within capacity

**Estimated overhead:**
- Current: ~7µs per publish (raw bytes)
- With JSON: ~10-12µs per publish (JSON envelope)
- **Impact: ~40% increase in latency, still well under 50ms target**

---

## Conclusion

### Direct Answer:
**NO, the tcp-conn frame format does NOT change.**

### What Actually Changes:
- The **content** of the payload (raw bytes → JSON envelopes)
- The **broker protocol** (simple pub/sub → delivery guarantees)
- The **application logic** (no acks → acks, no ordering → ordering)

### What Stays the Same:
- TCP-conn frame structure
- Connection multiplexing
- Chunking mechanism
- Connection pooling
- All tcp-conn APIs

### Recommendation:
Proceed with the spec implementation. The tcp-conn layer is unaffected and will continue to work exactly as it does now. The changes are purely at the broker/application layer.

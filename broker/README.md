# Broker Package

Pub/sub messaging broker built on top of the socket package.

## Key Concept

**The broker package is just socket clients orchestrated for pub/sub.** It doesn't modify the socket layer - it uses Consumer and Provider as-is to create a message distribution pattern.

**All connections are persistent** - no connection churn. Each subscriber maintains open TCP connections for the lifetime of the subscription.

## Architecture

### Connection Model

```
Broker Connections (persistent):
├── 1 Provider  (receives publishes on broker-chan)
└── 1 Consumer  (publishes to itself on broker-chan)

Per Subscriber (persistent):
├── 1 Provider  (receives forwarded messages on unique channel)
└── 1 Consumer  (broker uses to forward messages)

Total: 2 + 2N connections for N subscribers
```

### Message Flow

```
┌─────────────┐
│   Broker    │
│  Publish()  │
└──────┬──────┘
       │ Uses broker.publisher (persistent Consumer)
       ↓
┌─────────────────────────────────┐
│         Socket Server           │
└─────────────┬───────────────────┘
              ↓
       ┌──────────────┐
       │    Broker    │
       │  (Provider)  │ ← broker.provider (persistent)
       └──────┬───────┘
              │ Looks up topic → [chan-1, chan-2, chan-3]
              │ Uses broker.consumers[chanID] (persistent)
       ┌──────┴──────┬──────────────┐
       ↓             ↓              ↓
  ┌─────────┐  ┌─────────┐   ┌─────────┐
  │  Sub 1  │  │  Sub 2  │   │  Sub 3  │
  │(Provider│  │(Provider│   │(Provider│
  │ chan-1) │  │ chan-2) │   │ chan-3) │
  └─────────┘  └─────────┘   └─────────┘
     ↓             ↓              ↓
  Go channel   Go channel    Go channel
  (buffered)   (buffered)    (buffered)
```

## API

### Broker Methods

```go
// Create broker
broker, err := NewBroker(serverAddr, brokerChanID)

// Publish message to topic
err := broker.Publish(ctx, topic, payload, headers)

// Subscribe to topic (returns channel)
ch, err := broker.Subscribe(topic)

// Receive messages
msg := <-ch

// Unsubscribe from topic
broker.Unsubscribe(topic)

// Close broker
broker.Close()
```

### Message Structure

```go
type Message struct {
    Topic   string
    Payload []byte
    Headers map[string]string
}
```

## Usage Example

```go
package main

import (
    "context"
    "fmt"
    "pulsyflux/broker"
    "pulsyflux/socket"
    "github.com/google/uuid"
)

func main() {
    // Start socket server
    server := socket.NewServer("9090")
    server.Start()
    defer server.Stop()

    // Create broker
    brokerChanID := uuid.New()
    b, _ := broker.NewBroker("127.0.0.1:9090", brokerChanID)
    defer b.Close()

    // Subscribe to topic
    ch, _ := b.Subscribe("events")

    // Receive messages in goroutine
    go func() {
        for msg := range ch {
            fmt.Printf("Received: %s - %s\n", msg.Topic, string(msg.Payload))
        }
    }()

    // Publish messages
    ctx := context.Background()
    b.Publish(ctx, "events", []byte("Hello World"), nil)
    b.Publish(ctx, "events", []byte("Another message"), map[string]string{
        "priority": "high",
        "source":   "api",
    })
}
```

## Multiple Subscribers Example

```go
// Multiple subscribers to same topic
ch1, _ := broker.Subscribe("orders")
ch2, _ := broker.Subscribe("orders")
ch3, _ := broker.Subscribe("orders")

// All three receive the same message
broker.Publish(ctx, "orders", []byte("New order"), nil)

msg1 := <-ch1  // Receives message
msg2 := <-ch2  // Receives message
msg3 := <-ch3  // Receives message
```

## Multiple Topics Example

```go
// Subscribe to different topics
orders := broker.Subscribe("orders")
users := broker.Subscribe("users")
events := broker.Subscribe("events")

// Publish to specific topics
broker.Publish(ctx, "orders", []byte("Order #123"), nil)
broker.Publish(ctx, "users", []byte("User login"), nil)

// Each subscriber only receives their topic
orderMsg := <-orders  // Gets order message
userMsg := <-users    // Gets user message
```

## Connection Management

### Persistent Connections

Each `Subscribe()` call creates **2 persistent connections**:
1. **Provider** - receives forwarded messages from broker
2. **Consumer** - broker uses to forward messages to this subscriber

These connections remain open until:
- `Unsubscribe(topic)` is called
- `broker.Close()` is called
- Connection error occurs

### Connection Lifecycle

```go
// Subscribe creates persistent connections
ch, _ := broker.Subscribe("events")  // 2 connections created

// Connections stay open for message delivery
msg1 := <-ch  // Uses existing connections
msg2 := <-ch  // Uses existing connections
msg3 := <-ch  // Uses existing connections

// Unsubscribe closes connections
broker.Unsubscribe("events")  // 2 connections closed
```

### Resource Usage

For **N subscribers** on **M topics**:
- **Broker**: 2 connections (1 Provider + 1 Consumer)
- **Subscribers**: 2N connections (N Providers + N Consumers)
- **Total**: 2 + 2N TCP connections
- **All persistent** - no connection churn

Example with 100 subscribers:
- 2 broker connections
- 200 subscriber connections
- **202 total persistent connections**

## Performance Characteristics

### Latency
- Inherits socket package latency (~124µs)
- Additional overhead: JSON marshal/unmarshal + topic lookup
- Typical end-to-end: ~200-300µs

### Throughput
- Limited by broker's ability to forward messages
- Single broker can handle ~5K-8K messages/sec
- Scales with number of topics (independent forwarding)

### Memory
- Each subscriber channel buffered to 100 messages
- Persistent connections: ~8KB per connection
- For 100 subscribers: ~1.6MB connection overhead + message buffers

## Best Practices

### DO ✅

**Use buffered channels**
```go
// Channels are buffered (100) by default
ch, _ := broker.Subscribe("events")
// Can handle bursts without blocking broker
```

**Handle closed channels**
```go
for msg := range ch {
    // Process message
}
// Channel closed when unsubscribed
```

**Use context for timeouts**
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
broker.Publish(ctx, "events", data, nil)
```

### DON'T ❌

**Don't block on receive**
```go
// Bad - blocks forever if no messages
msg := <-ch

// Good - use select with timeout
select {
case msg := <-ch:
    process(msg)
case <-time.After(5 * time.Second):
    // Handle timeout
}
```

**Don't forget to unsubscribe**
```go
// Bad - leaks connections
ch, _ := broker.Subscribe("temp")
// ... use channel ...
// Forgot to unsubscribe!

// Good - clean up
ch, _ := broker.Subscribe("temp")
defer broker.Unsubscribe("temp")
```

**Don't create multiple brokers**
```go
// Bad - multiple brokers on same channel
b1, _ := broker.NewBroker(addr, chanID)
b2, _ := broker.NewBroker(addr, chanID)  // Conflict!

// Good - single broker instance
b, _ := broker.NewBroker(addr, chanID)
```

## Implementation Details

### How Forwarding Works

1. Publisher calls `Publish()` → uses `broker.publisher` (persistent Consumer)
2. Broker's `provider` receives message via `Receive()`
3. Broker looks up topic in `topics` map → gets list of channelIDs
4. For each channelID, broker uses `consumers[channelID]` (persistent Consumer)
5. Each subscriber's Provider receives via `Receive()` → sends to Go channel
6. Application reads from Go channel

### Why Two Connections Per Subscriber?

- **Provider**: Subscriber needs to receive messages (RPC server role)
- **Consumer**: Broker needs to send to subscriber (RPC client role)

This is the socket package's RPC model - Consumer sends requests, Provider receives them.

### Thread Safety

All broker methods are thread-safe:
- `Publish()` - can be called from multiple goroutines
- `Subscribe()` - can be called concurrently
- `Unsubscribe()` - protected by mutex
- Internal maps protected by `sync.RWMutex`

## Comparison with Alternatives

| Feature | Broker | NATS | Redis Pub/Sub |
|---------|--------|------|---------------|
| Setup | Simple | Medium | Simple |
| Dependencies | Socket only | NATS server | Redis server |
| Persistence | No | Optional | No |
| Latency | ~200µs | ~200µs | ~100µs |
| Throughput | ~5-8K msg/s | ~100K msg/s | ~50K msg/s |
| Connections | Persistent | Persistent | Persistent |
| Use Case | Internal services | Distributed systems | Cache + pub/sub |

## Limitations

1. **No message persistence** - messages lost if no subscribers
2. **Single broker** - no HA or clustering
3. **No wildcards** - exact topic match only
4. **In-memory only** - no disk storage
5. **No authentication** - relies on socket layer security

## Testing

```bash
cd broker
go test -v
```

All tests verify:
- Single and multiple subscribers
- Topic isolation
- Unsubscribe cleanup
- Connection persistence

# MessageBus Package

High-level pub/sub messaging system built on the socket transport layer.

## Overview

MessageBus provides a simple, efficient pub/sub messaging pattern for Go applications. It's built on top of the optimized socket package, inheriting its performance characteristics (123µs latency, 8K req/s throughput).

## Architecture

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│ Publisher   │         │   Server    │         │ Subscriber  │
│   Bus       │         │   (Broker)  │         │    Bus      │
│ Consumer────┼────────>│             ├────────>│──Provider   │
│             │         │   Routes    │         │             │
└─────────────┘         └─────────────┘         └─────────────┘
     Same Channel ID                                Same Channel ID
```

**Key Points:**
- Each Bus has both Consumer (publish) and Provider (subscribe)
- Publisher uses Consumer.Send() to send messages to server
- Server routes to all Providers on same channel
- Subscribers receive via Provider.Receive()
- Multiple buses on same channel = pub/sub broadcast

**Layers:**
```
MessageBus (pub/sub, topics, handlers)
    ↓
Socket (transport, framing, routing)
    ↓
TCP (network)
```

## Quick Start

### 1. Start Server

```go
server := messagebus.NewServer("9090")
server.Start()
defer server.Stop()
```

### 2. Create Bus

```go
channelID := uuid.New()
bus, err := messagebus.NewBus("127.0.0.1:9090", channelID)
if err != nil {
    log.Fatal(err)
}
defer bus.Close()
```

### 3. Subscribe

```go
ch, err := bus.Subscribe("orders.created")
if err != nil {
    log.Fatal(err)
}

go func() {
    for msg := range ch {
        log.Printf("Order: %s", string(msg.Payload))
    }
}()
```

### 4. Publish

```go
ctx := context.Background()
err := bus.Publish(ctx, "orders.created", []byte(`{"id": 123}`), nil)
```

## Core Concepts

### Message

```go
type Message struct {
    ID        uuid.UUID           // Unique message ID
    Topic     string              // Topic name (e.g., "orders.created")
    Payload   []byte              // Message data
    Headers   map[string]string   // Metadata
    Timestamp time.Time           // Creation time
}
```

### Topics

Topics are string identifiers for message routing:
- Use dot notation: `"orders.created"`, `"users.updated"`
- Case-sensitive
- No wildcards (yet)

### Handlers

Subscribe returns a channel for receiving messages:
```go
ch, err := bus.Subscribe("topic")
if err != nil {
    return err
}

// Receive messages
for msg := range ch {
    // Process message
}
```

Channels:
- Buffered with 100 message capacity
- Closed when topic is unsubscribed
- Non-blocking send (drops if full)

## Usage Examples

### Basic Pub/Sub

```go
package main

import (
    "context"
    "log"
    "pulsyflux/messagebus"
    "github.com/google/uuid"
)

func main() {
    // Start server
    server := messagebus.NewServer("9090")
    server.Start()
    defer server.Stop()

    channelID := uuid.New()

    // Create publisher bus
    pubBus, _ := messagebus.NewBus("127.0.0.1:9090", channelID)
    defer pubBus.Close()

    // Create subscriber bus
    subBus, _ := messagebus.NewBus("127.0.0.1:9090", channelID)
    defer subBus.Close()

    // Subscribe
    ch, _ := subBus.Subscribe("events")
    go func() {
        for msg := range ch {
            log.Printf("Received: %s", string(msg.Payload))
        }
    }()

    // Publish
    ctx := context.Background()
    pubBus.Publish(ctx, "events", []byte("Hello World"), nil)
}
```

### Multiple Subscribers

```go
// Multiple buses on same channel receive same message
channelID := uuid.New()

pubBus, _ := messagebus.NewBus(addr, channelID)
sub1, _ := messagebus.NewBus(addr, channelID)
sub2, _ := messagebus.NewBus(addr, channelID)

ch1, _ := sub1.Subscribe("orders")
ch2, _ := sub2.Subscribe("orders")

pubBus.Publish(ctx, "orders", data, nil)
// Both ch1 and ch2 receive the message
```

### With Headers

```go
// Publish with metadata
bus.Publish(ctx, "orders.created", orderData, map[string]string{
    "user_id":    "123",
    "source":     "api",
    "priority":   "high",
    "request_id": "abc-def",
})

// Access in subscriber
ch, _ := bus.Subscribe("orders.created")
for msg := range ch {
    userID := msg.Headers["user_id"]
    priority := msg.Headers["priority"]
    // Process...
}
```

### Error Handling

```go
bus.Subscribe("orders", func(ctx context.Context, msg *messagebus.Message) error {
    if err := processOrder(msg.Payload); err != nil {
        return err // Error sent back to publisher
    }
    return nil
})

// Publisher receives error
err := bus.Publish(ctx, "orders", data, nil)
if err != nil {
    log.Printf("Handler failed: %v", err)
}
```

### Context Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := bus.Publish(ctx, "slow.topic", data, nil)
if err == context.DeadlineExceeded {
    log.Println("Publish timed out")
}
```

### Unsubscribe

```go
bus.Subscribe("temp.topic", handler)

// Later...
bus.Unsubscribe("temp.topic") // Removes all handlers
```

## Use Cases

### 1. Event-Driven Microservices

```go
// Order Service
bus.Subscribe("orders.created", func(ctx context.Context, msg *messagebus.Message) error {
    return processNewOrder(msg.Payload)
})

// Inventory Service
bus.Subscribe("orders.created", func(ctx context.Context, msg *messagebus.Message) error {
    return reserveInventory(msg.Payload)
})

// Email Service
bus.Subscribe("orders.created", func(ctx context.Context, msg *messagebus.Message) error {
    return sendConfirmationEmail(msg.Payload)
})
```

### 2. Real-Time Notifications

```go
// Notification service
bus.Subscribe("user.activity", func(ctx context.Context, msg *messagebus.Message) error {
    userID := msg.Headers["user_id"]
    return pushNotification(userID, msg.Payload)
})

// Trigger from anywhere
bus.Publish(ctx, "user.activity", []byte("New message"), map[string]string{
    "user_id": "123",
})
```

### 3. Audit Logging

```go
// Central audit logger
bus.Subscribe("audit.*", func(ctx context.Context, msg *messagebus.Message) error {
    return logToDatabase(msg.Topic, msg.Payload, msg.Timestamp)
})

// Publish from services
bus.Publish(ctx, "audit.user.login", userData, nil)
bus.Publish(ctx, "audit.order.created", orderData, nil)
```

### 4. Data Pipeline

```go
// Stage 1: Ingest
bus.Subscribe("data.raw", func(ctx context.Context, msg *messagebus.Message) error {
    cleaned := cleanData(msg.Payload)
    return bus.Publish(ctx, "data.cleaned", cleaned, msg.Headers)
})

// Stage 2: Transform
bus.Subscribe("data.cleaned", func(ctx context.Context, msg *messagebus.Message) error {
    transformed := transform(msg.Payload)
    return bus.Publish(ctx, "data.transformed", transformed, msg.Headers)
})

// Stage 3: Store
bus.Subscribe("data.transformed", func(ctx context.Context, msg *messagebus.Message) error {
    return saveToDatabase(msg.Payload)
})
```

### 5. Background Jobs

```go
// Job worker
bus.Subscribe("jobs.email", func(ctx context.Context, msg *messagebus.Message) error {
    return sendEmail(msg.Payload)
})

bus.Subscribe("jobs.report", func(ctx context.Context, msg *messagebus.Message) error {
    return generateReport(msg.Payload)
})

// Enqueue jobs
bus.Publish(ctx, "jobs.email", emailData, nil)
bus.Publish(ctx, "jobs.report", reportData, nil)
```

## Best Practices

### ✅ DO

**1. Use Descriptive Topics**
```go
// Good
bus.Subscribe("orders.created", handler)
bus.Subscribe("users.password.reset", handler)

// Bad
bus.Subscribe("event1", handler)
bus.Subscribe("data", handler)
```

**2. Handle Errors Gracefully**
```go
bus.Subscribe("orders", func(ctx context.Context, msg *messagebus.Message) error {
    if err := process(msg); err != nil {
        log.Printf("Failed to process order %s: %v", msg.ID, err)
        return err
    }
    return nil
})
```

**3. Use Headers for Metadata**
```go
bus.Publish(ctx, "events", data, map[string]string{
    "trace_id":   traceID,
    "user_id":    userID,
    "source":     "api",
})
```

**4. Close Resources**
```go
bus, err := messagebus.NewBus(addr, channelID)
if err != nil {
    return err
}
defer bus.Close() // Always close
```

**5. Use Context for Timeouts**
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := bus.Publish(ctx, topic, data, nil)
```

### ❌ DON'T

**1. Don't Block in Handlers**
```go
// Bad
bus.Subscribe("events", func(ctx context.Context, msg *messagebus.Message) error {
    time.Sleep(10 * time.Second) // Blocks other messages
    return nil
})

// Good
bus.Subscribe("events", func(ctx context.Context, msg *messagebus.Message) error {
    go processAsync(msg) // Process in background
    return nil
})
```

**2. Don't Ignore Errors**
```go
// Bad
bus.Publish(ctx, topic, data, nil) // Ignored error

// Good
if err := bus.Publish(ctx, topic, data, nil); err != nil {
    log.Printf("Publish failed: %v", err)
}
```

**3. Don't Use Same Channel for Different Services**
```go
// Bad
orderBus, _ := messagebus.NewBus(addr, sharedChannelID)
userBus, _ := messagebus.NewBus(addr, sharedChannelID)
// Messages will cross-contaminate

// Good
orderBus, _ := messagebus.NewBus(addr, orderChannelID)
userBus, _ := messagebus.NewBus(addr, userChannelID)
```

**4. Don't Create Bus Per Message**
```go
// Bad
for _, msg := range messages {
    bus, _ := messagebus.NewBus(addr, channelID)
    bus.Publish(ctx, topic, msg, nil)
    bus.Close()
}

// Good
bus, _ := messagebus.NewBus(addr, channelID)
defer bus.Close()
for _, msg := range messages {
    bus.Publish(ctx, topic, msg, nil)
}
```

## Performance

### Characteristics
- **Latency**: ~123µs (inherited from socket layer)
- **Throughput**: ~8K messages/sec
- **Overhead**: JSON serialization + socket transport
- **Memory**: Minimal, streaming-based

### Optimization Tips

1. **Reuse Bus Instances**
   ```go
   // Create once, use many times
   bus, _ := messagebus.NewBus(addr, channelID)
   defer bus.Close()
   ```

2. **Batch When Possible**
   ```go
   // Send multiple items in one message
   batch := []Item{item1, item2, item3}
   data, _ := json.Marshal(batch)
   bus.Publish(ctx, topic, data, nil)
   ```

3. **Use Appropriate Timeouts**
   ```go
   // Short for simple operations
   ctx, _ := context.WithTimeout(ctx, 1*time.Second)
   
   // Longer for complex processing
   ctx, _ := context.WithTimeout(ctx, 30*time.Second)
   ```

## Comparison

| Feature | MessageBus | NATS | RabbitMQ |
|---------|-----------|------|----------|
| Pub/Sub | ✅ | ✅ | ✅ |
| Persistence | ❌ | ✅ | ✅ |
| Clustering | ❌ | ✅ | ✅ |
| Wildcards | ❌ | ✅ | ✅ |
| Latency | 123µs | ~200µs | ~1ms |
| Setup | Simple | Medium | Complex |
| Dependencies | None | NATS Server | RabbitMQ |

**When to use MessageBus:**
- Simple pub/sub needs
- Low latency required
- Minimal dependencies
- Internal microservices

**When to use alternatives:**
- Need persistence (RabbitMQ)
- Need clustering (NATS, RabbitMQ)
- Complex routing (RabbitMQ)
- Topic wildcards (NATS)

## Limitations

1. **No Persistence**: Messages not stored, lost if no subscribers
2. **No Wildcards**: Must subscribe to exact topic names
3. **No Clustering**: Single server, no HA
4. **No Authentication**: No built-in auth/authz
5. **No Message Replay**: Can't replay historical messages

## Roadmap

- [ ] Topic wildcards (`orders.*`, `users.*.created`)
- [ ] Message persistence (optional)
- [ ] Dead letter queue
- [ ] Message acknowledgments
- [ ] Metrics/observability
- [ ] Authentication
- [ ] TLS support
- [ ] Message replay
- [ ] Rate limiting

## Testing

```bash
go test -v ./messagebus
```

## License

MIT

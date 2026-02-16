# Broker Package (TCP-Conn Based)

A lightweight pub/sub broker built on top of the tcp-conn package.

## Architecture

```
┌─────────┐                    ┌─────────┐
│ Client  │───── Publish ─────>│ Server  │
│         │                    │         │
│         │<──── Subscribe ────│         │
└─────────┘                    └─────────┘
                                    │
                                    ├─> Topic A: [conn1, conn2]
                                    └─> Topic B: [conn3]
```

## Features

- **Connection Pooling**: Automatic via tcp-conn
- **Multiplexing**: Multiple logical connections over single TCP socket
- **Auto-Reconnect**: Client connections automatically reconnect
- **Topic-Based**: Publish/subscribe by topic name

## Usage

### Server

```go
server := broker.NewServer(":8080")
server.Start()
defer server.Stop()
```

### Client

```go
client, _ := broker.NewClient("localhost:8080")

// Publish
client.Publish("events", []byte("hello"))

// Subscribe
ch, _ := client.Subscribe("events")
for msg := range ch {
    fmt.Printf("Topic: %s, Data: %s\n", msg.Topic, string(msg.Payload))
}
```

## Message Flow

1. Client publishes message to topic
2. Server receives and broadcasts to all subscribers of that topic
3. Subscribers receive message via their channel

## Performance

Inherits tcp-conn performance characteristics:
- Low latency: ~20µs for small messages
- High throughput: Up to 630 MB/s for chunked transfers
- Connection pooling reduces overhead

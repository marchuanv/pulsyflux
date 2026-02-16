# Broker Package

A channel-based pub/sub message broker built on top of the tcp-conn package.

## Architecture

The broker uses a hierarchical connection model:

```
Client                          Server
┌─────────────────┐            ┌─────────────────┐
│ Control Conn    │───────────>│ Control Handler │
│ (random UUID)   │            │ (empty UUID)    │
└─────────────────┘            └─────────────────┘
        │                               │
        │ Sends Channel IDs             │ Creates Channels
        ▼                               ▼
┌─────────────────┐            ┌─────────────────┐
│ Channel Conn    │<──────────>│ Channel Handler │
│ (channel UUID)  │            │ (channel UUID)  │
└─────────────────┘            └─────────────────┘
```

### Connection Layers

1. **Control Connection**: Used to request channel creation
   - Client: Each client has unique random UUID
   - Server: Uses empty UUID to accept all control messages
   - Purpose: Send channel IDs to join/create channels

2. **Channel Connection**: Used for pub/sub messaging
   - Both sides use the same channel UUID
   - Multiple clients can connect to same channel
   - Messages broadcast to all channel members except sender

### Key Concepts

- **Channel**: A pub/sub topic identified by UUID
- **Logical Connection**: tcp-conn multiplexed connection
- **Physical Connection**: Underlying TCP socket (shared via tcp-conn pooling)

## API

### Server

```go
server := broker.NewServer(":8080")
server.Start()
defer server.Stop()

addr := server.Addr() // Get actual listening address
```

### Client

```go
client, err := broker.NewClient("localhost:8080")

// Publish to a channel
channelID := uuid.New()
client.Publish(channelID, "topic", []byte("message"))

// Subscribe to a channel
ch, err := client.Subscribe(channelID, "topic")
for msg := range ch {
    fmt.Printf("Topic: %s, Payload: %s\n", msg.Topic, string(msg.Payload))
}
```

## Message Structure

```go
type Message struct {
    Topic   string
    Payload []byte
}
```

Note: Topics are metadata only - all messages on a channel are broadcast to all subscribers regardless of topic.

## Usage Example

```go
package main

import (
    "fmt"
    "github.com/google/uuid"
    "pulsyflux/broker"
)

func main() {
    // Start server
    server := broker.NewServer(":0")
    server.Start()
    defer server.Stop()

    // Create clients
    client1, _ := broker.NewClient(server.Addr())
    client2, _ := broker.NewClient(server.Addr())

    // Both subscribe to same channel
    channelID := uuid.New()
    ch1, _ := client1.Subscribe(channelID, "events")
    ch2, _ := client2.Subscribe(channelID, "events")

    // Receive messages
    go func() {
        for msg := range ch2 {
            fmt.Printf("Client2 received: %s\n", string(msg.Payload))
        }
    }()

    // Publish message
    client1.Publish(channelID, "events", []byte("hello"))
    // Client2 receives "hello"
    // Client1 does NOT receive own message
}
```

## How It Works

### Client Side

1. **NewClient()**: Creates control connection with random UUID
2. **Subscribe()/Publish()**: 
   - Calls getChannel() which sends channel UUID on control connection
   - Creates channel connection with that UUID (reused if already exists)
   - Subscribe: Adds channel to subscriber list, returns Go channel
   - Publish: Sends message on channel connection

### Server Side

1. **Accept**: Server accepts TCP connection
2. **handleClient()**: Wraps connection with empty UUID control connection
   - Receives channel UUIDs from clients
   - Creates channel struct if new
   - Spawns handleChannel() for each unique (conn, channel) pair
3. **handleChannel()**: Wraps connection with channel UUID
   - Receives messages on channel connection
   - Broadcasts to all other connections in same channel

### Message Flow

```
Client1.Publish(channelID, "topic", data)
    │
    ├─> Channel Connection (channelID)
    │
    ▼
Server.handleChannel() receives message
    │
    ├─> Looks up channel by ID
    │
    ├─> Iterates through all connections in channel
    │
    ├─> Skips sender connection
    │
    ▼
Broadcasts to Client2, Client3, etc.
    │
    ▼
Client2.receiveLoop() receives message
    │
    ├─> Unmarshals JSON
    │
    ├─> Sends to all subscriber Go channels
    │
    ▼
Application receives from ch2
```

## Connection Multiplexing

The broker leverages tcp-conn's multiplexing:

- Multiple clients to same server share physical TCP connections
- Each logical connection has unique UUID for message routing
- Control and channel connections multiplex over same physical socket
- Reduces TCP handshake overhead and connection count

## Design Decisions

### Why Empty UUID on Server Control?

The server's control connection uses `uuid.UUID{}` (empty UUID) to accept messages from any client control connection. Each client uses a random UUID for its control connection to avoid conflicts when multiple clients share the same physical TCP connection via tcp-conn pooling.

### Why Not Use Topics for Filtering?

Currently, all messages on a channel are broadcast to all subscribers. Topics are included in the message but not used for filtering. This keeps the implementation simple. Topic-based filtering could be added at the application level or in a future version.

### Why Separate Control and Channel Connections?

- **Control**: Manages channel lifecycle (join/create)
- **Channel**: Handles high-frequency message traffic

This separation allows the control connection to handle multiple channel requests without blocking message flow.

## Performance Characteristics

- Inherits tcp-conn performance: ~20µs latency for small messages
- Connection pooling reduces overhead
- Multiplexing allows many logical connections over few physical sockets
- Broadcast is O(N) where N = number of subscribers in channel

## Limitations

- No message persistence
- No topic-based filtering (all channel subscribers receive all messages)
- Sender does not receive own messages
- No authentication/authorization
- Single server (no clustering)

## Testing

```bash
cd broker
go test -v
```

Tests verify:
- Basic pub/sub functionality
- Sender doesn't receive own messages
- Channel isolation (messages don't leak between channels)

## Future Enhancements

- Topic-based filtering
- Message persistence
- Wildcard topic subscriptions
- Authentication
- Clustering/HA
- Metrics and monitoring

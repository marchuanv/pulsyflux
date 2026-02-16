# Broker Package

A lightweight, channel-based pub/sub message broker built on tcp-conn multiplexing.

## Overview

The broker provides a simple pub/sub messaging system where:
- Clients connect to channels identified by UUID
- Messages published to a channel are broadcast to all other clients on that channel
- Senders do not receive their own messages
- Multiple channels are isolated from each other

## Quick Start

### Server

```go
server := broker.NewServer(":8080")
if err := server.Start(); err != nil {
    log.Fatal(err)
}
defer server.Stop()

addr := server.Addr() // Get actual listening address
```

### Client

```go
channelID := uuid.New()
client, err := broker.NewClient("localhost:8080", channelID)
if err != nil {
    log.Fatal(err)
}

// Subscribe to receive messages
ch := client.Subscribe()
go func() {
    for msg := range ch {
        fmt.Printf("Received: %s\n", string(msg))
    }
}()

// Publish messages
client.Publish([]byte("hello world"))
```

## Complete Example

```go
package main

import (
    "fmt"
    "github.com/google/uuid"
    "pulsyflux/broker"
    "time"
)

func main() {
    // Start server
    server := broker.NewServer(":0")
    server.Start()
    defer server.Stop()

    // Create channel
    channelID := uuid.New()

    // Create two clients on same channel
    client1, _ := broker.NewClient(server.Addr(), channelID)
    client2, _ := broker.NewClient(server.Addr(), channelID)

    // Subscribe both clients
    ch1 := client1.Subscribe()
    ch2 := client2.Subscribe()

    // Receive messages
    go func() {
        for msg := range ch2 {
            fmt.Printf("Client2 received: %s\n", string(msg))
        }
    }()

    time.Sleep(100 * time.Millisecond)

    // Publish from client1
    client1.Publish([]byte("hello"))
    // Output: Client2 received: hello
    // Note: Client1 does NOT receive its own message
}
```

## API Reference

### Server

#### `NewServer(address string) *Server`
Creates a new broker server.
- `address`: TCP address to listen on (e.g., ":8080" or "localhost:8080")
- Use ":0" to let OS assign a random port

#### `(*Server) Start() error`
Starts the server and begins accepting connections.
- Returns error if unable to bind to address
- Non-blocking - runs accept loop in goroutine

#### `(*Server) Addr() string`
Returns the actual listening address.
- Useful when using ":0" to get the assigned port

#### `(*Server) Stop() error`
Stops the server and closes the listener.
- Closes all active connections
- Should be called with defer after Start()

### Client

#### `NewClient(address string, channelID uuid.UUID) (*Client, error)`
Creates a new client connected to a specific channel.
- `address`: Server address (e.g., "localhost:8080")
- `channelID`: UUID identifying the channel to join
- Automatically sends control message to server
- Establishes channel connection for pub/sub
- Returns error if connection fails

#### `(*Client) Subscribe() <-chan []byte`
Creates a subscription to receive messages from the channel.
- Returns a buffered channel (capacity 100)
- Multiple subscriptions can be created per client
- All subscriptions receive all messages on the channel
- Non-blocking - messages are dropped if channel is full

#### `(*Client) Publish(payload []byte) error`
Publishes a message to the channel.
- `payload`: Raw bytes to send
- Message is broadcast to all other clients on the channel
- Sender does NOT receive their own message
- Returns error if send fails

## Architecture

### Connection Model

The broker uses tcp-conn's multiplexing to efficiently manage connections:

```
Multiple Clients                    Server
┌─────────────────┐                ┌─────────────────┐
│ Client 1        │                │                 │
│ UUID: client-1  │───┐            │                 │
└─────────────────┘   │            │                 │
                      ├──> Shared  │  Demultiplexer  │
┌─────────────────┐   │    Physical│  Routes by UUID │
│ Client 2        │───┤    TCP     │                 │
│ UUID: client-2  │   │    Socket  │                 │
└─────────────────┘   │            │                 │
                      │            │                 │
┌─────────────────┐   │            │                 │
│ Client 3        │───┘            │                 │
│ UUID: client-3  │                │                 │
└─────────────────┘                └─────────────────┘
```

### Connection Layers

**1. Physical Layer**
- Single TCP socket shared by multiple clients
- Managed by tcp-conn connection pooling
- Reduces connection overhead

**2. Control Layer**
- Uses GlobalControlUUID (00000000-0000-0000-0000-000000000000)
- Purpose: Initialize channel connections
- Client sends: `{ClientID: "uuid", ChannelID: "uuid"}`
- Server creates channel connection for that client
- Used ONLY during client construction

**3. Channel Layer**
- Each client has unique UUID for identification
- Messages multiplexed over shared physical connection
- Server routes messages by client UUID
- All pub/sub happens here

### Message Flow

```
Client1.Publish(data)
    │
    ├─> Send on channel connection (clientID)
    │
    ▼
Server receives on logical connection
    │
    ├─> Lookup channel by channelID
    │
    ├─> Iterate all clients in channel
    │
    ├─> Skip sender (clientID)
    │
    ▼
Broadcast to Client2, Client3, etc.
    │
    ▼
Client2.receiveLoop() receives data
    │
    ├─> Send to all subscriber channels
    │
    ▼
Application receives from Subscribe()
```

## Implementation Details

### Server

**Data Structures:**
```go
type Server struct {
    address  string                           // Listen address
    listener net.Listener                     // TCP listener
    channels map[uuid.UUID]*channel           // channelID -> channel
    clients  map[uuid.UUID]net.Conn          // clientID -> connection
    mu       sync.RWMutex                     // Protects channels/clients
    done     chan struct{}                    // Shutdown signal
}

type channel struct {
    clients map[uuid.UUID]*tcpconn.Connection // clientID -> logical conn
    mu      sync.RWMutex                       // Protects clients map
}
```

**Control Message:**
```go
type controlMessage struct {
    ClientID  string `json:"client_id"`   // Client UUID
    ChannelID string `json:"channel_id"`  // Channel UUID
}
```

**Flow:**
1. `acceptLoop()` accepts TCP connections
2. `handleClient()` wraps connection with GlobalControlUUID
3. Receives control messages with ClientID + ChannelID
4. Creates/gets channel struct
5. Wraps connection with ClientID for that channel
6. Spawns `handleChannel()` goroutine
7. `handleChannel()` receives messages and broadcasts to other clients

### Client

**Data Structures:**
```go
type Client struct {
    id        uuid.UUID              // Unique client ID
    channelID uuid.UUID              // Channel this client is on
    address   string                 // Server address
    conn      *tcpconn.Connection    // Channel connection
    subs      []chan []byte          // Subscriber channels
    mu        sync.RWMutex           // Protects subs
}
```

**Flow:**
1. `NewClient()` generates random clientID
2. Creates control connection with GlobalControlUUID
3. Sends controlMessage with clientID + channelID
4. Creates channel connection with clientID
5. Starts `receiveLoop()` goroutine
6. `Subscribe()` adds channel to subs list
7. `Publish()` sends directly on channel connection
8. `receiveLoop()` distributes received messages to all subs

## Design Decisions

### Idle Timeout Behavior

The broker inherits tcp-conn's 30-second idle timeout:
- Logical connections close after 30 seconds without activity
- Physical connections close when all logical connections are closed
- Clients should send periodic messages to maintain connections
- Automatic reconnection occurs on next operation (may cause message loss)

### Why Channel ID in Constructor?

Each client is bound to a single channel for its lifetime. This simplifies the API and makes the client's purpose clear. To communicate on multiple channels, create multiple clients.

### Why No Topics?

Topics add complexity without clear benefit for this use case. Channel isolation provides sufficient message routing. Applications can implement their own message filtering if needed.

### Why Raw Bytes?

The broker is transport-agnostic. Applications can use JSON, protobuf, msgpack, or any serialization format. This keeps the broker simple and flexible.

### Why Skip Sender?

Prevents message loops and simplifies application logic. The sender already knows what they published. If echo behavior is needed, applications can implement it.

### Why Connection Pooling?

tcp-conn's multiplexing allows many logical connections over few physical sockets. This reduces:
- TCP handshake overhead
- File descriptor usage
- Network resource consumption
- Connection establishment latency

## Performance

- **Latency**: ~7µs for publish, ~41µs for round-trip pub/sub
- **Throughput**: Limited by network bandwidth and broadcast fanout
- **Scalability**: O(N) broadcast where N = clients per channel
- **Memory**: Minimal - no message buffering or persistence
- **Idle timeout**: 30 seconds (from tcp-conn) - connections close if inactive

## Limitations

- No message persistence or replay
- No message acknowledgment or delivery guarantees
- No authentication or authorization
- No encryption (use TLS proxy if needed)
- Single server (no clustering or HA)
- Sender cannot receive own messages
- No backpressure - slow subscribers drop messages
- **Idle timeout**: Logical connections close after 30 seconds of inactivity (inherited from tcp-conn)
  - Physical connection closes when all logical connections are closed
  - Clients must send periodic messages or implement keepalive to maintain connections
  - Reconnection is automatic but may cause message loss during reconnection

## Testing

```bash
cd broker
go test -v
```

**Test Coverage:**
- `TestBasicPubSub`: Verifies pub/sub and sender exclusion
- `TestMultipleChannels`: Verifies channel isolation
- `TestDebug`: Manual debugging test with verbose output

## Thread Safety

All public methods are thread-safe:
- Server methods can be called from multiple goroutines
- Client methods can be called from multiple goroutines
- Multiple Subscribe() calls are safe
- Concurrent Publish() calls are safe

## Error Handling

**Server:**
- `Start()` returns error if bind fails
- Connection errors are logged and ignored
- Malformed control messages are ignored

**Client:**
- `NewClient()` returns error if connection fails
- `Publish()` returns error if send fails
- Subscribe never errors (returns channel immediately)

## Future Enhancements

- Message acknowledgment
- Delivery guarantees (at-least-once, exactly-once)
- Message persistence and replay
- Authentication and authorization
- TLS support
- Metrics and monitoring
- Clustering and high availability
- Backpressure handling

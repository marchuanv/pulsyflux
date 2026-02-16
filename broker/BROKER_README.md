# Broker Package

A channel-based pub/sub message broker built on top of the tcp-conn package.

## Design Rules

### CRITICAL: Connection Pooling is Required

**Multiple clients MUST share the same physical TCP connection via tcp-conn's connection pooling.**

This is NOT a bug - this is the entire point of the connection pool:
- Multiple clients connecting to the same broker address will share one physical TCP socket
- The broker MUST distinguish clients using logical connections (UUIDs), NOT by net.Conn
- Each client has a unique client UUID for identification
- Each channel has a unique channel UUID for message routing
- The tcp-conn multiplexing handles routing messages to the correct logical connection

**DO NOT:**
- Try to force separate physical connections per client
- Use net.Conn as a client identifier
- Assume one physical connection = one client

**DO:**
- Use client UUIDs to identify clients
- Use channel UUIDs to route messages
- Leverage tcp-conn's multiplexing for efficiency

## Architecture

The broker uses a hierarchical connection model with multiplexing:

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

1. **Physical Layer**: Single TCP socket shared by multiple clients (via tcp-conn pooling)
2. **Control Layer**: Global control UUID (00000000-0000-0000-0000-000000000000) for establishing channels
   - **PURPOSE**: ONLY for initialization - establishing new channel connections
   - **USAGE**: Client sends control message with ClientID + ChannelID to request channel creation
   - **NOT FOR**: Message exchange, pub/sub, or any data transfer
   - All clients send control messages on the same logical connection (GlobalControlUUID)
   - Server receives control messages and creates channel connections for that client
3. **Channel Layer**: Each channel has its own UUID for message routing
   - **PURPOSE**: ALL message exchange happens here (pub/sub)
   - **USAGE**: Client publishes/receives messages on channel-specific logical connections
   - Messages are multiplexed over the shared physical connection
   - tcp-conn's demux routes messages by UUID to the correct logical connection

### Control vs Channel Connections

**Control Connection (GlobalControlUUID):**
```go
// ONLY used once per channel to initialize
controlMsg := clientMessage{
    ClientID:  "client-uuid",
    ChannelID: "channel-uuid",
}
control.Send(controlMsg) // Tells server: "Create channel connection for this client"
```

**Channel Connection (ChannelUUID):**
```go
// Used for ALL message exchange
msg := Message{
    Topic:   "events",
    Payload: []byte("data"),
}
channelConn.Send(msg) // Actual pub/sub messages
```

**CRITICAL RULE**: Never send pub/sub messages on control connection. Control is ONLY for initialization.

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

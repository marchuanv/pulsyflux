# Socket Package

A reliable and flexible message bus system with per-client request/response queues.

## Architecture

The socket package implements a server-side client registry that manages multiple concurrent clients with strict FIFO ordering.

### Core Components

- **Server**: Accepts connections and routes frames based on flags
- **Client**: Provides Send(), Receive(), and Respond() operations
- **Registry**: Manages per-client request and response queues

### Frame Flags

- `flagRequest (0x01)`: Request frames from Send()
- `flagReceive (0x04)`: Signal to dequeue a request
- `flagResponse (0x08)`: Response frames from Respond()

## Usage

### Server
```go
server := NewServer("8080")
server.Start()
defer server.Stop()
```

### Client
```go
client, _ := NewClient("127.0.0.1:8080", channelID)
defer client.Close()

// Send a request
response, _ := client.Send(reader, timeout)

// Receive a request
request, _ := client.Receive()

// Respond to a request
client.Respond(reader, timeout)
```

## Frame Routing

### Send()
- Client sends frames with `flagRequest`
- Server enqueues to all peers' request queues in same channel
- If no peers, enqueues to sender's own request queue

### Receive()
- Client sends frame with `flagReceive`
- Server dequeues from client's own request queue first
- If empty, dequeues from peers' request queues
- Server sends dequeued frame directly to client

### Respond()
- Client sends frames with `flagResponse`
- Server routes to original requester's response queue
- Response queue processor sends frames in FIFO order

## Files

- `server.go` - Server implementation
- `client.go` - Client implementation
- `registry.go` - Client registry with queues
- `connctx.go` - Connection context and frame I/O
- `shared.go` - Shared constants and errors
- `bufferpool.go` - Buffer pooling
- `specification.md` - Detailed specification
- `context.md` - Architecture documentation

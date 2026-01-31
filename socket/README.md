# Socket Package

A bidirectional request-response socket system with a server acting as a message broker between consumers and providers on specific channels.

## Architecture

```
Consumer → Server → Provider
         ↓        ↓
    Registry   Workers
         ↓        ↓
    Provider → Server → Consumer
```

## Key Components

### 1. Server (`server.go`)
- TCP server that brokers messages between consumers and providers
- Maintains a client registry to track connections by channel ID
- Uses a pool of 64 request workers to handle message routing
- Handles connection lifecycle and frame forwarding
- Configurable socket buffers (512KB send/receive)

### 2. Consumer (`consumer.go`)
- Sends requests to providers via the server
- Blocks waiting for response with configurable timeout
- Uses streaming chunks for large payloads
- Auto-generates unique client ID per instance

### 3. Provider (`provider.go`)
- Registers a handler function to process requests
- Listens for incoming requests in a goroutine
- Sends responses back through the server
- Maintains routing info for response delivery

### 4. Frame Protocol (`frame.go`)
- Binary protocol with 24-byte header:
  - Version (1 byte)
  - Type (1 byte)
  - Flags (2 bytes)
  - RequestID (16 bytes UUID)
  - Payload length (4 bytes)
- Frame types: `StartFrame`, `ChunkFrame`, `EndFrame`, `ResponseFrame`, `ErrorFrame`
- Max frame size: 1MB
- Read timeout: 2 minutes
- Write timeout: 5 seconds

### 5. Client Registry (`clientreg.go`)
- Tracks consumers and providers by `channelID` and `clientID`
- Provides peer lookup for routing messages
- Thread-safe with RWMutex
- Auto-cleanup on client disconnect

### 6. Connection Context (`connctx.go`)
- Manages async writes per connection with buffered channels
- Prioritizes error frames over regular frames
- Runs a dedicated writer goroutine per connection
- Write buffer: 8192 frames
- Error buffer: 2048 frames

### 7. Request Workers (`reqworker.go`, `reqhandler.go`)
- Pool of 64 workers processing requests from a queue
- Queue size: 8192 requests
- Forwards frames between peers with routing info (clientID + channelID)
- Handles peer disappearance gracefully

### 8. Base Client (`baseclient.go`)
- Shared functionality for Consumer and Provider
- Handles connection dialing and registration
- Builds metadata payloads
- Assembles chunked responses

## Message Flow

1. **Connection**: Consumer/Provider connects and sends `StartFrame` with metadata (role, timeout, clientID, channelID)
2. **Registration**: Server registers client in registry
3. **Request**: Consumer sends `StartFrame` → `ChunkFrame(s)` → `EndFrame`
4. **Routing**: Server forwards to matching provider on same channelID
5. **Processing**: Provider processes request and sends `ResponseFrame`/`ErrorFrame`
6. **Response**: Server routes response back to consumer using routing info

## Routing Info Format

32-byte routing header prepended to payloads:
- Bytes 0-15: Client ID (UUID)
- Bytes 16-31: Channel ID (UUID)

## Key Features

- **Multiple consumers per provider**: Load distribution across consumers
- **Concurrent channels**: Isolated communication channels via UUID
- **Large payload streaming**: Automatic chunking for payloads > 1MB
- **Multi-level timeouts**: Connection, request, and frame-level timeouts
- **Graceful cleanup**: Proper connection and resource cleanup
- **Backpressure handling**: Drops frames when buffers full
- **Error prioritization**: Error frames bypass regular queue

## Usage Example

```go
// Start server
server := NewServer("9090")
server.Start()
defer server.Stop()

// Create provider
channelID := uuid.New()
provider, _ := NewProvider("127.0.0.1:9090", channelID, func(payload []byte) ([]byte, error) {
    return []byte("processed: " + string(payload)), nil
})
defer provider.Close()

// Create consumer
consumer, _ := NewConsumer("127.0.0.1:9090", channelID)
defer consumer.Close()

// Send request
response, _ := consumer.Send(strings.NewReader("hello"), 5*time.Second)
fmt.Println(string(response)) // "processed: hello"
```

## Configuration

- **Worker pool size**: 64 workers
- **Worker queue size**: 8192 requests
- **Write buffer**: 8192 frames per connection
- **Error buffer**: 2048 frames per connection
- **Socket buffers**: 512KB send/receive
- **Max frame size**: 1MB
- **Default timeout**: 5 seconds
- **Frame read timeout**: 2 minutes
- **Frame write timeout**: 5 seconds

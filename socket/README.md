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
- Uses one request handler managing 64 worker goroutines for message routing
- Each worker runs concurrently processing requests from its own queue
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

### 7. Request Handler & Workers (`reqworker.go`, `reqhandler.go`)
- One request handler managing 64 worker goroutines running concurrently
- Each worker has its own dedicated queue (128 requests per worker, 8192 total)
- **RequestID-based routing**: Frames for the same RequestID are hashed to the same worker
- Ensures sequential processing of StartFrame → ChunkFrame(s) → EndFrame per request
- Workers process different requests in parallel across CPU cores
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
- **Frame ordering**: RequestID-based worker routing ensures frames are processed in order
- **Concurrent request processing**: Different requests can be processed in parallel by different workers

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

- **Worker goroutines**: 64 concurrent workers
- **Per-worker queue size**: 128 requests (8192 total capacity)
- **Write buffer**: 8192 frames per connection
- **Error buffer**: 2048 frames per connection
- **Socket buffers**: 512KB send/receive
- **Max frame size**: 1MB
- **Default timeout**: 5 seconds
- **Frame read timeout**: 2 minutes
- **Frame write timeout**: 5 seconds

## Implementation Details

### Frame Ordering

The system ensures frames for the same request are processed in order:

1. **Server receives frames** from consumer in order: StartFrame → ChunkFrame(s) → EndFrame
2. **RequestID hashing** routes all frames for the same RequestID to the same worker
3. **Worker processes sequentially** from its dedicated queue
4. **Async writes** through connctx preserve order on the wire

This design allows:
- **Sequential processing** of frames within a request
- **Parallel processing** of different requests across workers
- **No race conditions** between frames of the same request

## Performance

### Benchmarks (Intel i5-12400F, 12 cores)

| Benchmark | Requests/sec | Latency | Memory/op |
|-----------|--------------|---------|------------|
| Single Request | 3,333 | 300µs | 1.0 MB |
| Concurrent (10 consumers) | 8,680 | 115µs | 1.0 MB |
| Large Payload (1MB) | 860 | 1.16ms | 4.0 MB |
| Multiple Channels (10) | 9,916 | 100µs | 1.0 MB |
| High Throughput (100 consumers) | 11,560 | 86µs | 1.0 MB |

### Resource Management

- **No goroutine leaks**: Verified across 10 iterations
- **No memory leaks**: <0.5 MB growth after 100 iterations
- **Clean registry**: All connections properly cleaned up
- **Throughput**: ~860 MB/s for large payloads
- **Scalability**: 2.6x performance improvement with concurrency

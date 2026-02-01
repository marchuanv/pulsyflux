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

## Detailed Lifecycle & Message Flow

```
┌─────────────┐                    ┌─────────────┐                    ┌─────────────┐
│  Consumer   │                    │   Server    │                    │  Provider   │
└──────┬──────┘                    └──────┬──────┘                    └──────┬──────┘
       │                                  │                                  │
       │ 1. Connect & Register            │                                  │
       ├─────────────────────────────────>│                                  │
       │   StartFrame (FlagRegistration)  │                                  │
       │   - clientID, channelID, role    │                                  │
       │                                  │                                  │
       │                                  │ 2. Add to Registry               │
       │                                  │    (consumers map)               │
       │                                  │                                  │
       │                                  │ 3. Connect & Register            │
       │                                  │<─────────────────────────────────┤
       │                                  │   StartFrame (FlagRegistration)  │
       │                                  │   - clientID, channelID, role    │
       │                                  │                                  │
       │                                  │ 4. Add to Registry               │
       │                                  │    (providers map)               │
       │                                  │                                  │
       │ 5. Send Request                  │                                  │
       ├─────────────────────────────────>│                                  │
       │   StartFrame (metadata)          │                                  │
       │                                  │                                  │
       │                                  │ 6. Hash RequestID → Worker       │
       │                                  │    Queue frames to Worker N      │
       │                                  │                                  │
       │                                  │ 7. Forward StartFrame            │
       │                                  ├─────────────────────────────────>│
       │                                  │   (routing info: clientID +      │
       │                                  │    channelID prepended)          │
       │                                  │                                  │
       │ 8. Send Chunk(s)                 │                                  │
       ├─────────────────────────────────>│                                  │
       │   ChunkFrame (payload data)      │                                  │
       │                                  │                                  │
       │                                  │ 9. Same Worker (RequestID hash)  │
       │                                  │                                  │
       │                                  │ 10. Forward ChunkFrame           │
       │                                  ├─────────────────────────────────>│
       │                                  │                                  │
       │                                  │                                  │
       │ 11. Send End                     │                                  │
       ├─────────────────────────────────>│                                  │
       │   EndFrame                       │                                  │
       │                                  │                                  │
       │                                  │ 12. Same Worker (RequestID hash) │
       │                                  │                                  │
       │                                  │ 13. Forward EndFrame             │
       │                                  ├─────────────────────────────────>│
       │                                  │                                  │
       │                                  │                                  │ 14. Process Request
       │                                  │                                  │     handler(payload)
       │                                  │                                  │
       │                                  │ 15. Send ResponseStartFrame      │
       │                                  │<─────────────────────────────────┤
       │                                  │   (routing info in payload)      │
       │                                  │                                  │
       │                                  │ 16. Extract routing, track       │
       │                                  │     consumer for RequestID       │
       │                                  │                                  │
       │                                  │ 17. Send ResponseChunkFrame(s)   │
       │                                  │<─────────────────────────────────┤
       │                                  │   (response data chunks)         │
       │                                  │                                  │
       │                                  │ 18. Forward chunks to consumer   │
       │                                  │     using tracked connection     │
       │                                  │                                  │
       │                                  │ 19. Send ResponseEndFrame        │
       │                                  │<─────────────────────────────────┤
       │                                  │                                  │
       │                                  │ 20. Forward end, cleanup         │
       │                                  │     RequestID tracking           │
       │                                  │                                  │
       │ 21. Receive ResponseStartFrame   │                                  │
       │<─────────────────────────────────┤                                  │
       │                                  │                                  │
       │ 22. Receive ResponseChunkFrame(s)│                                  │
       │<─────────────────────────────────┤                                  │
       │   Assemble payload               │                                  │
       │                                  │                                  │
       │ 23. Receive ResponseEndFrame     │                                  │
       │<─────────────────────────────────┤                                  │
       │   Return complete io.Reader      │                                  │
       │                                  │                                  │
       │ 24. Close                        │                                  │
       ├─────────────────────────────────>│                                  │
       │                                  │                                  │
       │                                  │ 25. Remove from Registry         │
       │                                  │                                  │
       │                                  │ 26. Close                        │
       │                                  │<─────────────────────────────────┤
       │                                  │                                  │
       │                                  │ 27. Remove from Registry         │
       │                                  │                                  │
```

**Key Points:**
- Steps 6, 9, 12: RequestID hashing ensures same worker processes all frames
- Steps 7, 10, 13: Frames forwarded in order by single worker
- Step 14: Provider processes complete request after receiving all frames
- Steps 15-20: Provider sends chunked response, server tracks consumer connection
- Steps 21-23: Consumer receives and assembles chunked response

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
- `Send(r io.Reader, timeout time.Duration)` returns `(io.Reader, error)`
- Blocks waiting for response with configurable timeout
- Uses streaming chunks for large payloads (both request and response)
- Auto-generates unique client ID per instance

### 3. Provider (`provider.go`)
- Channel-based API for receiving requests and sending responses
- `Receive()` returns `(uuid.UUID, io.Reader, bool)` for getting requests
- `Respond(reqID uuid.UUID, r io.Reader, err error)` for sending responses
- Listens for incoming requests in a goroutine
- Sends chunked responses back through the server
- Maintains routing info for response delivery

### 4. Frame Protocol (`frame.go`)
- Binary protocol with 24-byte header:
  - Version (1 byte)
  - Type (1 byte)
  - Flags (2 bytes)
  - RequestID (16 bytes UUID)
  - Payload length (4 bytes)
- Frame types:
  - **Request frames**: `StartFrame`, `ChunkFrame`, `EndFrame`
  - **Response frames**: `ResponseStartFrame`, `ResponseChunkFrame`, `ResponseEndFrame`
  - **Other**: `ResponseFrame` (legacy), `ErrorFrame`
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
- `sendChunkedRequest()` - sends request chunks
- `sendChunkedResponse()` - sends response chunks with routing info
- `receiveChunkedResponse()` - assembles chunked responses

## Message Flow

### Request Flow (Consumer → Provider)
1. **Connection**: Consumer/Provider connects and sends `StartFrame` with metadata (role, timeout, clientID, channelID)
2. **Registration**: Server registers client in registry
3. **Request**: Consumer sends `StartFrame` → `ChunkFrame(s)` → `EndFrame`
4. **Routing**: Server forwards to matching provider on same channelID with routing info prepended
5. **Processing**: Provider receives request via `Receive()` and processes it

### Response Flow (Provider → Consumer)
1. **Response**: Provider calls `Respond()` with data or error
2. **Chunking**: Provider sends `ResponseStartFrame` (with routing info) → `ResponseChunkFrame(s)` → `ResponseEndFrame`
3. **Routing**: Server tracks consumer connection from `ResponseStartFrame` and forwards chunks
4. **Assembly**: Consumer assembles chunks and returns complete response as `io.Reader`

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
server := socket.NewServer("9090")
server.Start()
defer server.Stop()

// Create provider
channelID := uuid.New()
provider, _ := socket.NewProvider("127.0.0.1:9090", channelID)
defer provider.Close()

// Handle requests in background
go func() {
    for {
        reqID, r, ok := provider.Receive()
        if !ok {
            break
        }
        data, _ := io.ReadAll(r)
        provider.Respond(reqID, bytes.NewReader([]byte("processed: "+string(data))), nil)
    }
}()

// Create consumer
consumer, _ := socket.NewConsumer("127.0.0.1:9090", channelID)
defer consumer.Close()

// Send request
response, _ := consumer.Send(strings.NewReader("hello"), 5*time.Second)
data, _ := io.ReadAll(response)
fmt.Println(string(data)) // "processed: hello"
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

### Chunked Responses

Responses are also chunked for efficient large payload handling:

1. **Provider sends**: ResponseStartFrame (with routing info) → ResponseChunkFrame(s) → ResponseEndFrame
2. **Server tracks**: Extracts routing from ResponseStartFrame, stores consumer connection per RequestID
3. **Server forwards**: Routes ResponseChunkFrame and ResponseEndFrame to correct consumer
4. **Consumer assembles**: Collects all chunks and returns complete payload as io.Reader

Benefits:
- Symmetric chunking for both requests and responses
- No single-frame size limits for responses
- Efficient memory usage with streaming

## Performance

### Benchmarks (Intel i5-12400F, 12 cores)

| Benchmark | Requests/sec | Latency | Memory/op |
|-----------|--------------|---------|------------|
| Single Request | 2,062 | 485µs | 2.1 MB |
| Concurrent (10 consumers) | 3,151 | 336µs | 2.1 MB |
| Large Payload (1MB) | 674 | 1.48ms | 5.2 MB |
| Multiple Channels (10) | 5,015 | 199µs | 2.1 MB |
| High Throughput (100 consumers) | 3,794 | 264µs | 2.1 MB |

### Resource Management

- **No goroutine leaks**: Verified across 10 iterations
- **No memory leaks**: <0.5 MB growth after 100 iterations
- **Clean registry**: All connections properly cleaned up
- **Throughput**: ~675 MB/s for large payloads
- **Scalability**: Improved performance with concurrency


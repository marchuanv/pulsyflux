# Socket Package

A bidirectional request-response socket system with a server acting as a message broker between paired clients on specific channels.

## Architecture

```
Client1 ←→ Server ←→ Client2
           ↓
      Worker Pool (64)
           ↓
      Peer Registry
```

## Handshake Flow

```
┌─────────┐                 ┌─────────┐                 ┌─────────┐
│ Client1 │                 │ Server  │                 │ Client2 │
└────┬────┘                 └────┬────┘                 └────┬────┘
     │                           │                           │
     │ 1. Connect TCP            │                           │
     ├──────────────────────────>│                           │
     │                           │                           │
     │ 2. START (flagHandshake)  │                           │
     ├──────────────────────────>│                           │
     │   clientID: C1            │                           │
     │   channelID: CH1          │                           │
     │                           │                           │
     │                           │ 3. Register C1            │
     │                           │    peers[C1] = {CH1, ctx} │
     │                           │                           │
     │                           │ 4. Lookup peer on CH1     │
     │                           │    → none found           │
     │                           │                           │
     │                           │    5. Connect TCP         │
     │                           │<──────────────────────────┤
     │                           │                           │
     │                           │ 6. START (flagHandshake)  │
     │                           │<──────────────────────────┤
     │                           │   clientID: C2            │
     │                           │   channelID: CH1          │
     │                           │                           │
     │                           │ 7. Register C2            │
     │                           │    peers[C2] = {CH1, ctx} │
     │                           │                           │
     │                           │ 8. Lookup peer on CH1     │
     │                           │    → found C1!            │
     │                           │                           │
     │ 9. START (flagHandshake)  │                           │
     │<──────────────────────────┤                           │
     │   clientID: C2 (peer)     │                           │
     │   peerClientID: C1        │                           │
     │                           │                           │
     │                           │ 10. START (flagHandshake) │
     │                           ├──────────────────────────>│
     │                           │    clientID: C1 (peer)    │
     │                           │    peerClientID: C2       │
     │                           │                           │
     │ ✓ Paired with C2          │                           │
     │                           │          ✓ Paired with C1 │
     │                           │                           │
```

## Bidirectional Message Flow

```
┌─────────┐                 ┌─────────┐                 ┌─────────┐
│ Client1 │                 │ Server  │                 │ Client2 │
└────┬────┘                 └────┬────┘                 └────┬────┘
     │                           │                           │
     │ Send("hello")             │                           │
     │                           │                           │
     │ 1. START                  │                           │
     ├──────────────────────────>│                           │
     │   reqID: R1               │                           │
     │   peerClientID: C2        │                           │
     │                           │                           │
     │                           │ 2. Hash R1 → Worker 23    │
     │                           │    Queue to worker        │
     │                           │                           │
     │                           │ 3. Worker routes START    │
     │                           ├──────────────────────────>│
     │                           │   reqID: R1               │
     │                           │                           │
     │                           │                           │ Respond() waiting
     │                           │                           │ receives START
     │                           │                           │
     │                           │ 4. START response         │
     │                           │<──────────────────────────┤
     │                           │   reqID: R1               │
     │                           │                           │
     │                           │ 5. Route back to C1       │
     │<──────────────────────────┤                           │
     │   (START ack)             │                           │
     │                           │                           │
     │ 2. CHUNK                  │                           │
     ├──────────────────────────>│                           │
     │   reqID: R1               │                           │
     │   payload: "hello"        │                           │
     │                           │                           │
     │                           │ 6. Hash R1 → Worker 23    │
     │                           │    (same worker!)         │
     │                           │                           │
     │                           │ 7. Worker routes CHUNK    │
     │                           ├──────────────────────────>│
     │                           │   payload: "hello"        │
     │                           │                           │
     │                           │                           │ Read "hello"
     │                           │                           │ Write "echo"
     │                           │                           │
     │                           │ 8. CHUNK response         │
     │                           │<──────────────────────────┤
     │                           │   reqID: R1               │
     │                           │   payload: "echo"         │
     │                           │                           │
     │                           │ 9. Route back to C1       │
     │<──────────────────────────┤                           │
     │   payload: "echo"         │                           │
     │                           │                           │
     │ 3. END                    │                           │
     ├──────────────────────────>│                           │
     │   reqID: R1               │                           │
     │                           │                           │
     │                           │ 10. Hash R1 → Worker 23   │
     │                           │                           │
     │                           │ 11. Worker routes END     │
     │                           ├──────────────────────────>│
     │                           │                           │
     │                           │                           │ Detect END
     │                           │                           │ Send END response
     │                           │                           │
     │                           │ 12. END response          │
     │                           │<──────────────────────────┤
     │                           │   reqID: R1               │
     │                           │                           │
     │                           │ 13. Route back to C1      │
     │<──────────────────────────┤                           │
     │                           │                           │
     │ Return io.Reader("echo")  │                           │
     │                           │                           │
```

## Key Components

### Server (`server.go`)
- TCP server broking messages between paired clients
- Maintains peer registry by channelID
- 64 worker goroutines for async frame routing
- Handles handshake pairing and frame forwarding

### Client (`client.go`)
- `Send(r io.Reader, timeout) (io.Reader, error)` - sends request, waits for response
- `Respond(r io.Reader, timeout) error` - waits for request, sends response
- Bidirectional: both clients can Send() and Respond()

### Frame Protocol (`connctx.go`)
- 80-byte header: version, type, flags, requestID, clientID, peerClientID, channelID, timeout, payload length
- Types: START, CHUNK, END, ERROR
- Flags: flagHandshake (0x01), flagNone (0x00)
- Max payload: 1MB per frame

### Worker Pool (`reqhandler.go`, `reqworker.go`)
- 64 workers, each with dedicated queue (128 requests)
- RequestID hashing ensures same worker processes all frames for a request
- Sequential processing per request, parallel across requests
- Routes frames between peers with context timeout management

### Peer Registry (`peers.go`)
- Tracks clients by clientID and channelID
- `pair(clientID, channelID)` finds peer on same channel
- Thread-safe with RWMutex

## Frame Ordering Guarantee

1. Client sends: START → CHUNK(s) → END
2. Server hashes requestID to worker (e.g., R1 → Worker 23)
3. All frames for R1 go to Worker 23's queue
4. Worker 23 processes sequentially: START, then CHUNK, then END
5. Peer receives frames in order

Different requests (R1, R2, R3) can be processed in parallel by different workers.

## Usage

```go
// Start server
server := NewServer("9090")
server.Start()
defer server.Stop()

// Create paired clients
channelID := uuid.New()
client1, _ := NewClient("127.0.0.1:9090", channelID)
client2, _ := NewClient("127.0.0.1:9090", channelID)
defer client1.Close()
defer client2.Close()

// Client2 responds in background
go func() {
    client2.Respond(strings.NewReader("echo"), 5*time.Second)
}()

// Client1 sends request
response, _ := client1.Send(strings.NewReader("hello"), 5*time.Second)
data, _ := io.ReadAll(response)
fmt.Println(string(data)) // "echo"
```

## Configuration

- Workers: 64
- Queue per worker: 128 requests
- Write buffer: 8192 frames/connection
- Read buffer: 1000 frames/connection
- Error buffer: 2048 frames/connection
- Socket buffers: 2MB send/receive
- Default timeout: 5 seconds

## Performance

- Latency: ~125µs per request
- Throughput: ~8,000 req/s (single client)
- Memory: ~7KB per request
- Allocations: ~79 per request

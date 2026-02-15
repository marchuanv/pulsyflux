# Socket Package Specification

## Status: IMPLEMENTED

This document describes the socket package implementation - a TCP-based pub-sub (broadcast) message bus with acknowledgment-based synchronization.

## Architecture

The socket layer is a **broadcast-only pub-sub system**:
- `Stream(in io.Reader, out io.Writer, onComplete func(error))` - unified async method that handles send, receive, and respond patterns
  - `in != nil, out == nil` - send mode: broadcasts message to ALL other clients on channel (excludes sender)
  - `in == nil, out != nil` - receive mode: receives broadcasts from other clients, writes payload to out
  - `in != nil, out != nil` - respond mode: receives broadcast, writes request to out, broadcasts response from in
- Server broadcasts frames to ALL clients EXCEPT the sender
- Clients don't need to filter out their own messages (server handles exclusion)
- Session is recreated on each Stream call to prevent unconsumed message issues

## Frame Protocol

### Frame Types
- `errorFrame (0x03)` - Error notification
- `startFrame (0x04)` - Beginning of a message
- `chunkFrame (0x05)` - Payload data chunks (up to 1MB per chunk)
- `endFrame (0x06)` - End of a message
- `ackFrame (0x07)` - Acknowledgment that a data frame was received

### Frame Flags
- `flagNone (0x00)` - No flags (used for registration frame)
- `flagRequest (0x01)` - Data frame being broadcast
- `flagAck (0x02)` - Control frame - acknowledgment

### Frame Header (84 bytes)
```
Bytes 0-1:    Version (1) + Type (1)
Bytes 2-3:    Flags (2)
Bytes 4-19:   RequestID (16 UUID)
Bytes 20-35:  ClientID (16 UUID)
Bytes 36-51:  ChannelID (16 UUID)
Bytes 52-67:  FrameID (16 UUID) - for ack tracking
Bytes 68-75:  ClientTimeoutMs (8)
Bytes 76-79:  Sequence (4) - chunk index + isFinal flag
Bytes 80-83:  PayloadLength (4)
```

### Frame Categories

**Data Frames** (application-level):
- `startFrame` - signals beginning, no payload, uses `flagRequest`
- `chunkFrame` - carries application data in payload, uses `flagRequest`
- `endFrame` - signals end, no payload, uses `flagRequest`

**Control Frames** (protocol-level):
- `ackFrame` - acknowledges receipt, no payload, uses `flagAck`
- `errorFrame` - reports errors, payload contains error message, uses `flagNone`

**Key Rule**: Application data ONLY goes in chunkFrame payload. Control frames never carry application data (except error messages).

## Client Registration

Registration is **implicit** - server registers client on first frame received:

```
Client                  Server
  |
  +--NewClient()--------->|
  | (connect TCP)         |
  | (start reader/writer) |
  |                       |
  +--registration frame-->|
  | (flagNone)            +--registry.register()
  |                       | (implicit registration)
  |<--registration ack----+
  | (flagNone)            | (send ack with flagNone)
  | (client ready)        |
```

The client sends a registration frame (startFrame with flagNone) immediately after connecting. Server registers the client and sends back an ack frame with flagNone. Client waits for this ack with 2-second timeout before becoming ready.

## Frame-by-Frame Acknowledgment Flow

Each frame requires acknowledgment from ALL receivers before sender proceeds:

```
Sender                  Server                  Receiver
  |
  +--startFrame---------->|  (DATA FRAME)
  | flagRequest           +--broadcast------------>|
  |                       |                         |
  | (blocked)             |                         +--ackFrame---------->
  |                       |<------------------------| (CONTROL FRAME)
  |                       | (aggregate)             | flagAck
  |<--ackFrame------------+                         |
  | (CONTROL FRAME)       |                         |
  | (unblocked)           |                         |
```

**Process**:
1. Sender sends frame with `flagRequest`
2. Server broadcasts to all receivers (excludes sender)
3. Each receiver sends `ackFrame` with `flagAck`
4. Server collects acks (N receivers → N acks)
5. Server sends single ack to sender
6. Sender proceeds to next frame

## Server Implementation

### Components

**Server** (`server.go`):
- Listens on TCP with SO_REUSEADDR and 2MB buffers
- One goroutine per connection in `handle()`
- Routes frames based on flags:
  - `flagRequest` → broadcast to peers via `handleRequest()`
  - `flagAck` → record ack via `registry.recordAck()`
  - Other → discard

**Registry** (`registry.go`):
- Maps `clientID → clientEntry` for direct lookup
- Maps `channelID → set of clientEntry` for channel routing
- Tracks pending acks via `pendingAcks map[uuid.UUID]*ackCollector`
- Each `clientEntry` has:
  - `requestQueue` - incoming broadcasts (buffered 1024)
  - `responseQueue` - outgoing acks/errors (buffered 1024)
  - Background goroutines to drain queues

**AckCollector**:
```go
type ackCollector struct {
    frameID       uuid.UUID  // Unique ID for this frame instance
    requestID     uuid.UUID  // The message requestID
    senderID      uuid.UUID  // Who sent the frame
    frameType     byte       // start/chunk/end
    expectedAcks  int        // Number of receivers
    receivedAcks  int        // Acks received so far
    done          chan struct{}  // Signal when all acks received
    mu            sync.Mutex
}
```

### Server Routing Logic

**handleRequest** (for `flagRequest` frames):
1. Call `waitForReceivers()` to get peer list (excludes sender)
2. If no receivers after timeout → send error frame "no receivers available"
3. If receivers available:
   - Generate unique `FrameID`
   - Create `ackCollector` expecting N acks
   - Broadcast frame to all N receivers
   - Wait on `<-collector.done` OR timeout (using `ClientTimeoutMs`)
   - On success: send ack to sender
   - On timeout: send error frame "timeout waiting for acknowledgments"
   - Cleanup collector

**waitForReceivers**:
- Uses notification-based approach (no polling)
- Immediately returns peer list if receivers exist
- If no receivers, registers notification channel in `channelNotify` map
- Blocks on notification channel until receiver joins or timeout
- When new receiver registers, all waiting senders are notified via their channels
- Returns peer list when notified and receivers exist
- Returns nil on timeout or context cancellation

**recordAck** (for `flagAck` frames):
- Finds ackCollector by FrameID
- Increments receivedAcks counter
- If receivedAcks == expectedAcks → close `done` channel

## Client Implementation

### API Methods

**Stream(in io.Reader, out io.Writer, onComplete func(error))**:
```go
func (c *Client) Stream(in io.Reader, out io.Writer, onComplete func(error)) {
    // Async - returns immediately
    // Recreates session to prevent unconsumed message issues
    // Spawns goroutine that:
    //   - If in != nil && out != nil: waits for incoming request, writes to out, sends response from in
    //   - If in != nil && out == nil: sends message from in (broadcast)
    //   - If in == nil && out != nil: waits for incoming request, writes to out (receive)
    // Calls onComplete(error) when operation finishes
    // If onComplete is nil, uses no-op function
}
```

**Usage Patterns**:
```go
// Send only
client.Stream(strings.NewReader("hello"), nil, func(err error) {
    if err != nil {
        log.Printf("Send failed: %v", err)
    }
})

// Receive only
var buf bytes.Buffer
client.Stream(nil, &buf, func(err error) {
    if err != nil {
        log.Printf("Receive failed: %v", err)
    }
    fmt.Println(buf.String())
})

// Respond (request-response)
var reqBuf bytes.Buffer
client.Stream(bytes.NewReader(respData), &reqBuf, func(err error) {
    if err != nil {
        log.Printf("Respond failed: %v", err)
    }
})
```

### Helper Methods

**sendFrame** - consolidated method for sending any frame type:
- Creates frame with specified type, flags, payload
- Sets sequence field (index + isFinal flag)
- Sends to `ctx.writes` channel

**receiveFrame** - consolidated method for receiving any frame type:
- Filters frames by RequestID and Type
- Returns matching frame or error
- Discards non-matching frames

**waitForAck**:
- Reads from `responses` channel (routed by routeFrames)
- Waits for `ackFrame` with matching RequestID
- Handles `errorFrame` (extracts error message from payload)
- Returns error on timeout or closed connection

**sendAck**:
- Sends `ackFrame` with `flagAck`
- Includes FrameID and RequestID for correlation

**routeFrames** (background goroutine):
- Reads from `ctx.reads` channel
- Routes frames based on `flagRequest`:
  - Frames WITH `flagRequest` → `requests` channel
  - Frames WITHOUT `flagRequest` → `responses` channel

**processIncoming** (background goroutine):
- Reads from `requests` channel
- Receives start frame, sends ack immediately
- Receives chunk frames, sends acks immediately, assembles payload
- Receives end frame, sends ack immediately
- Queues assembled request to `session.incoming` channel
- Acknowledgments sent before consumer reads from `session.incoming`

### Concurrency Control

**Session Management**:
- Session recreated on each `Stream()` call with buffered `incoming` channel (1024)
- Prevents unconsumed message issues by checking for pending messages before recreating
- Returns error "session has unconsumed messages" if previous session has unread data
- Each Stream operation gets a fresh session

**Callback Pattern**:
- `Stream()` is async and returns immediately
- Spawns goroutine to handle the operation
- Calls `onComplete(error)` callback when operation finishes
- If onComplete is nil, uses no-op function to prevent panics
- Background goroutines handle frame routing and request assembly

**Connection Context** (`connctx`):
- Separate reader/writer goroutines per connection
- Channels:
  - `writes` - outgoing frames (buffered 1024)
  - `reads` - incoming frames (buffered 1024)
  - `errors` - priority error frames (buffered 256)
  - `closed` - shutdown signal
- Writer prioritizes error frames
- Read timeout: 2 minutes per frame
- Write timeout: 5 seconds per frame

**Client Channels**:
- `requests` - incoming request frames with `flagRequest` (buffered 1024)
- `responses` - ack/error frames without `flagRequest` (buffered 1024)
- `session.incoming` - assembled requests ready for consumption (buffered 1024)
- `registered` - unbuffered channel for registration ack signal
- `done` - unbuffered channel for shutdown signal

**Background Goroutines**:
- `routeFrames()` - routes frames from `ctx.reads` to `requests` or `responses`, handles registration ack
- `processIncoming()` - assembles incoming requests and queues to `incoming`
- `idleMonitor()` - monitors client activity and closes connection after 60 seconds of inactivity

## Buffer Pooling

- **Frame pool** (`sync.Pool`) - reuses frame structs
- **Buffer pool** (`sync.Pool`) - 1MB buffers for chunk reading
- **Write buffer pool** (`sync.Pool`) - 8KB+header buffers for small writes
- Frames returned via `putFrame()` after processing
- Buffers returned via `putBuffer()` / `putWriteBuffer()`

## Error Handling

**Error Types**:
- `errClosed` - client/connection closed
- `errFrame` - error frame received
- `errInvalidFrame` - wrong frame type received
- `errTimeout` - operation timeout

**Error Frame Flow**:
- Server sends error frame with `flagNone`
- Payload contains error message string
- Client receives in `waitForAck()` and returns error

**No Receivers Scenario**:
1. Client calls `Send()`
2. Server waits for receivers (polls every 100ms)
3. On timeout → server sends error frame "no receivers available"
4. Client's `waitForAck()` receives error frame
5. `Send()` returns error to caller

**Slow Receivers Scenario**:
1. Client calls `Send()`
2. Server finds receivers and broadcasts frame
3. Server waits for acks with timeout (using `ClientTimeoutMs`)
4. If not all acks received within timeout → server sends error frame "timeout waiting for acknowledgments"
5. Client's `waitForAck()` receives error frame
6. `Send()` returns error to caller

## Timeouts

- **Frame read timeout**: 2 minutes (`defaultFrameReadTimeout`)
- **Frame write timeout**: 5 seconds (`defaultFrameWriteTimeout`)
- **Client timeout**: 5 seconds default (`defaultTimeout`)
- **ClientTimeoutMs**: 30000ms default in frame header
- **Registration timeout**: 2 seconds (client waits for server registration ack)
- **Idle timeout**: 60 seconds (client closes after inactivity)
- **Idle check interval**: 5 seconds (idleMonitor ticker)

## Usage Examples

**Send only**:
```go
client, _ := NewClient("127.0.0.1:9090", channelID)
// No explicit close - idleMonitor handles lifecycle

var wg sync.WaitGroup
wg.Add(1)
client.Stream(strings.NewReader("hello"), nil, func(err error) {
    if err != nil {
        log.Printf("Send failed: %v", err)
    }
    wg.Done()
})
wg.Wait() // Block until send completes
```

**Receive only**:
```go
client, _ := NewClient("127.0.0.1:9090", channelID)

var buf bytes.Buffer
var wg sync.WaitGroup
wg.Add(1)
client.Stream(nil, &buf, func(err error) {
    if err != nil {
        log.Printf("Receive failed: %v", err)
    }
    fmt.Println(buf.String())
    wg.Done()
})
wg.Wait() // Block until receive completes
```

**Respond (request-response)**:
```go
client, _ := NewClient("127.0.0.1:9090", channelID)

var reqBuf bytes.Buffer
respData := []byte("response")
var wg sync.WaitGroup
wg.Add(1)
client.Stream(bytes.NewReader(respData), &reqBuf, func(err error) {
    if err != nil {
        log.Printf("Respond failed: %v", err)
    }
    wg.Done()
})
wg.Wait() // Block until respond completes
```

**Multiple transactions** (simplified with nil callback):
```go
client, _ := NewClient("127.0.0.1:9090", channelID)

for i := 0; i < 5; i++ {
    client.Stream(strings.NewReader(fmt.Sprintf("msg%d", i)), nil, nil)
}
time.Sleep(100 * time.Millisecond) // Allow operations to complete
```

## Key Implementation Details

1. **Implicit Registration**: Client sends registration frame (startFrame with flagNone) on connect, waits for ack with 2s timeout
2. **Registration Ack**: Server sends ack frame with flagNone after registration, client detects via `routeFrames()`
3. **Server Excludes Sender**: `getChannelPeersLocked()` excludes sender from peer list
4. **Frame-by-Frame Sync**: Sender blocks after each frame until all acks received
5. **Ack Aggregation**: N receivers → N acks → 1 ack to sender
6. **FrameID Tracking**: Each broadcast frame gets unique FrameID for ack correlation
7. **RequestID Purpose**: Groups frames together (start/chunk/end) and enables frame filtering
8. **Sequence Field**: Chunk index (31 bits) + isFinal flag (1 bit) using `sequenceFinalFlag = 0x80000000`
9. **Unified API**: Single `Stream()` method handles send, receive, and respond patterns based on in/out parameters
10. **Callback Pattern**: Operations use `onComplete(error)` callback instead of Wait() pattern
11. **Frame Routing**: `routeFrames()` separates request frames from response frames, handles registration ack
12. **No Client-Side Filtering**: Server handles sender exclusion
13. **Consolidated Frame Methods**: Single `sendFrame()` for all frame types
14. **Error in Payload**: Error frames carry error message in payload field
15. **Immediate Acknowledgment**: `processIncoming()` sends acks immediately upon receiving frames, not waiting for consumer
16. **Session Recreation**: Session recreated on each `Stream()` call to prevent unconsumed message issues
17. **Idle Management**: `idleMonitor()` closes client after 60s inactivity, checks every 5s
18. **Activity Tracking**: `updateActivity()` called on each Stream() to reset idle timer
19. **No Public Close**: Client lifecycle managed by idleMonitor, no public Close() method
20. **Channel Notification**: Registry uses `channelNotify` map to wake up senders when receivers join
21. **TCP Optimizations**: NoDelay enabled, 2MB read/write buffers on both client and server

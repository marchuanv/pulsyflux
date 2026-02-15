# Socket Package Specification

## Status: IMPLEMENTED

This document describes the socket package implementation - a TCP-based pub-sub (broadcast) message bus with acknowledgment-based synchronization.

## Architecture

The socket layer is a **broadcast-only pub-sub system**:
- `Send(r io.Reader) error` - broadcasts message to ALL other clients on channel (excludes sender)
- `Receive(incoming chan io.Reader) error` - receives broadcasts from other clients
- `Respond(incoming, outgoing chan io.Reader) error` - receives broadcasts, processes, and broadcasts response
- Server broadcasts frames to ALL clients EXCEPT the sender
- Clients don't need to filter out their own messages (server handles exclusion)

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
- `flagResponse (0x08)` - Control frame - used for errors

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
- `errorFrame` - reports errors, payload contains error message, uses `flagResponse`

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
  | (client ready)        |
```

The client sends a registration frame (startFrame with flagNone) immediately after connecting to trigger server-side registration.

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
  - `flagResponse` → route to specific client
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
    frameType     uint8      // start/chunk/end
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
   - Block on `<-collector.done` until all acks received
   - Send ack to sender
   - Cleanup collector

**waitForReceivers**:
- Polls every 100ms for receivers
- Returns peer list when found
- Returns nil on timeout

**recordAck** (for `flagAck` frames):
- Finds ackCollector by FrameID
- Increments receivedAcks counter
- If receivedAcks == expectedAcks → close `done` channel

## Client Implementation

### API Methods

**Send(r io.Reader) error**:
```go
func (c *Client) Send(r io.Reader) error {
    c.opMu.Lock()  // Serialize operations
    defer c.opMu.Unlock()

    reqID := uuid.New()
    
    // Send start + wait for ack
    sendFrame(startFrame, reqID, ...)
    waitForAck(reqID)
    
    // Send chunks + wait for ack after each
    for each chunk {
        sendFrame(chunkFrame, reqID, payload, ...)
        waitForAck(reqID)
    }
    
    // Send end + wait for ack
    sendFrame(endFrame, reqID, ...)
    waitForAck(reqID)
    
    return nil
}
```

**Receive(incoming chan io.Reader) error**:
```go
func (c *Client) Receive(incoming chan io.Reader) error {
    c.opMu.Lock()  // Serialize operations
    defer c.opMu.Unlock()

    // Receive start + send ack
    startF := receiveFrame(startFrame, uuid.Nil)
    sendAck(startF.FrameID, startF.RequestID)
    
    // Receive chunks + send ack after each
    for each chunk {
        chunkF := receiveFrame(chunkFrame, reqID)
        sendAck(chunkF.FrameID, reqID)
        append to buffer
        if isFinal break
    }
    
    // Receive end + send ack
    endF := receiveFrame(endFrame, reqID)
    sendAck(endF.FrameID, reqID)
    
    incoming <- bytes.NewReader(buffer)
    return nil
}
```

**Respond(incoming, outgoing chan io.Reader) error**:
```go
func (c *Client) Respond(incoming, outgoing chan io.Reader) error {
    Receive(incoming)  // Receive and ack
    reader := <-outgoing  // Wait for response
    if reader != nil {
        Send(reader)  // Broadcast response
    }
    return nil
}
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
- Waits for `ackFrame` with matching RequestID
- Handles `errorFrame` (extracts error message from payload)
- Returns error on timeout or closed connection

**sendAck**:
- Sends `ackFrame` with `flagAck`
- Includes FrameID and RequestID for correlation

### Concurrency Control

**opMu Lock**:
- Serializes `Send()`, `Receive()`, `Respond()` on same client
- Prevents frame stealing from shared `ctx.reads` channel
- Concurrent calls block and execute sequentially (not rejected)

**Connection Context** (`connctx`):
- Separate reader/writer goroutines per connection
- Channels:
  - `writes` - outgoing frames (buffered 1024)
  - `reads` - incoming frames (buffered 1024)
  - `errors` - priority error frames (buffered 512)
  - `closed` - shutdown signal
- Writer prioritizes error frames
- Read timeout: 2 minutes per frame
- Write timeout: 5 seconds per frame

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
- Server sends error frame with `flagResponse`
- Payload contains error message string
- Client receives in `waitForAck()` and returns error

**No Receivers Scenario**:
1. Client calls `Send()`
2. Server waits for receivers (polls every 100ms)
3. On timeout → server sends error frame "no receivers available"
4. Client's `waitForAck()` receives error frame
5. `Send()` returns error to caller

## Timeouts

- **Frame read timeout**: 2 minutes (`defaultFrameReadTimeout`)
- **Frame write timeout**: 5 seconds (`defaultFrameWriteTimeout`)
- **Client timeout**: 5 seconds default (`defaultTimeout`)
- **ClientTimeoutMs**: 30000ms default in frame header
- **Receiver wait polling**: 100ms interval

## Usage Examples

**Send only**:
```go
client, _ := NewClient("127.0.0.1:9090", channelID)
defer client.Close()

err := client.Send(strings.NewReader("hello"))
```

**Receive only**:
```go
client, _ := NewClient("127.0.0.1:9090", channelID)
defer client.Close()

incoming := make(chan io.Reader, 1)
go func() {
    msg := <-incoming
    data, _ := io.ReadAll(msg)
    fmt.Println(string(data))
}()

client.Receive(incoming)
```

**Respond (request-response)**:
```go
client, _ := NewClient("127.0.0.1:9090", channelID)
defer client.Close()

incoming := make(chan io.Reader, 1)
outgoing := make(chan io.Reader, 1)

go func() {
    req := <-incoming
    data, _ := io.ReadAll(req)
    response := process(data)
    outgoing <- bytes.NewReader(response)
}()

client.Respond(incoming, outgoing)
```

## Key Implementation Details

1. **Implicit Registration**: Client sends registration frame (flagNone) on connect
2. **Server Excludes Sender**: `getChannelPeers()` excludes sender from peer list
3. **Frame-by-Frame Sync**: Sender blocks after each frame until all acks received
4. **Ack Aggregation**: N receivers → N acks → 1 ack to sender
5. **FrameID Tracking**: Each broadcast frame gets unique FrameID for ack correlation
6. **Sequence Field**: Chunk index (31 bits) + isFinal flag (1 bit)
7. **Operation Serialization**: `opMu` prevents concurrent operations on same client
8. **No Client-Side Filtering**: Server handles sender exclusion
9. **Consolidated Frame Methods**: Single `sendFrame()` and `receiveFrame()` for all types
10. **Error in Payload**: Error frames carry error message in payload field

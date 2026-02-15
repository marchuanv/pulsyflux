# Socket Package Specification

## Status: IMPLEMENTED

This document describes the socket package implementation - a TCP-based pub-sub (broadcast) message bus with acknowledgment-based synchronization.

## Architecture

The socket layer is a **broadcast-only pub-sub system**:
- `Send(r io.Reader)` - async, broadcasts message to ALL other clients on channel (excludes sender)
- `Receive(r io.Reader)` - async, receives broadcasts from other clients, writes payload to r (must be io.Writer)
- `Respond(req io.Reader, resp io.Reader)` - async, receives broadcasts, writes request to req, broadcasts response
- `Wait() error` - blocks until all async operations complete, returns first error
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
- Polls every 100ms for receivers
- Returns peer list when found
- Returns nil on timeout

**recordAck** (for `flagAck` frames):
- Finds ackCollector by FrameID
- Increments receivedAcks counter
- If receivedAcks == expectedAcks → close `done` channel

## Client Implementation

### API Methods

**Send(r io.Reader)**:
```go
func (c *Client) Send(r io.Reader) {
    // Async - returns immediately
    // Spawns goroutine that:
    //   - Sends start frame + waits for ack
    //   - Sends chunk frames + waits for ack after each
    //   - Sends end frame + waits for ack
    // Errors captured in opErrors channel
}
```

**Receive(r io.Reader)**:
```go
func (c *Client) Receive(r io.Reader) {
    // Async - returns immediately
    // Spawns goroutine that:
    //   - Reads from pre-assembled incoming channel
    //   - Writes payload to r (must be io.Writer)
    // Errors captured in opErrors channel
}
```

**Respond(req io.Reader, resp io.Reader)**:
```go
func (c *Client) Respond(req io.Reader, resp io.Reader) {
    // Async - returns immediately
    // Spawns goroutine that:
    //   - Reads from pre-assembled incoming channel
    //   - Writes request payload to req (must be io.Writer)
    //   - Calls Send(resp) if resp != nil
    // Errors captured in opErrors channel
}
```

**Wait() error**:
```go
func (c *Client) Wait() error {
    // Transactional - can be called multiple times
    // Blocks until all pending async operations complete
    // Checks for unconsumed messages in session (returns error if found)
    // Ends current session and creates new session for next operations
    // Drains opErrors channel and returns first error
    // Returns nil if no errors
    // After Wait() returns, new operations can be started with fresh session
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
- Session created at client initialization with buffered `incoming` channel (1024)
- Session persists across multiple Send/Receive/Respond operations
- `Wait()` ends current session and creates new session
- `Wait()` validates no unconsumed messages remain (returns error if found)
- Operations can be called in any order (Send, Receive, Respond)
- Multiple Receive/Respond can run concurrently reading from same session

**Transactional Wait Pattern**:
- `Send()`, `Receive()`, `Respond()` are all async and return immediately
- All operations tracked via `opWg` (sync.WaitGroup)
- Errors collected in `opErrors` channel (buffered 100)
- `Wait()` blocks until all operations complete and returns first error
- `Wait()` can be called multiple times - each call waits for pending operations
- After `Wait()` returns, new operations can be started
- Background goroutines handle frame routing and request assembly

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

**Client Channels**:
- `requests` - incoming request frames with `flagRequest` (buffered 1024)
- `responses` - ack/error frames without `flagRequest` (buffered 1024)
- `session.incoming` - assembled requests ready for consumption (buffered 1024)

**Background Goroutines**:
- `routeFrames()` - routes frames from `ctx.reads` to `requests` or `responses`
- `processIncoming()` - assembles incoming requests and queues to `incoming`

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
- **Receiver wait polling**: 100ms interval

## Usage Examples

**Send only**:
```go
client, _ := NewClient("127.0.0.1:9090", channelID)
defer client.Close()

client.Send(strings.NewReader("hello"))
err := client.Wait() // Block until send completes
```

**Receive only**:
```go
client, _ := NewClient("127.0.0.1:9090", channelID)
defer client.Close()

var buf bytes.Buffer
client.Receive(&buf)
client.Wait() // Block until receive completes
fmt.Println(buf.String())
```

**Respond (request-response)**:
```go
client, _ := NewClient("127.0.0.1:9090", channelID)
defer client.Close()

var reqBuf bytes.Buffer
respData := process(reqBuf.Bytes())
client.Respond(&reqBuf, bytes.NewReader(respData))
client.Wait() // Block until respond completes
```

**Multiple transactions**:
```go
client, _ := NewClient("127.0.0.1:9090", channelID)
defer client.Close()

for i := 0; i < 5; i++ {
    client.Send(strings.NewReader(fmt.Sprintf("msg%d", i)))
    client.Wait() // Wait for this send to complete
}
```

## Key Implementation Details

1. **Implicit Registration**: Client sends registration frame (startFrame with flagNone) on connect
2. **Server Excludes Sender**: `getChannelPeers()` excludes sender from peer list
3. **Frame-by-Frame Sync**: Sender blocks after each frame until all acks received
4. **Ack Aggregation**: N receivers → N acks → 1 ack to sender
5. **FrameID Tracking**: Each broadcast frame gets unique FrameID for ack correlation
6. **RequestID Purpose**: Groups frames together (start/chunk/end) and enables frame filtering
7. **Sequence Field**: Chunk index (31 bits) + isFinal flag (1 bit)
8. **Async Operations**: All operations (`Send`, `Receive`, `Respond`) are async and return immediately
9. **Operation Tracking**: `opWg` tracks all async operations, `Wait()` blocks until complete
10. **Frame Routing**: `routeFrames()` separates request frames from response frames
11. **No Client-Side Filtering**: Server handles sender exclusion
12. **Consolidated Frame Methods**: Single `sendFrame()` and `receiveFrame()` for all types
13. **Error in Payload**: Error frames carry error message in payload field
14. **Immediate Acknowledgment**: `processIncoming()` sends acks immediately upon receiving frames, not waiting for consumer
15. **Transactional Wait**: `Wait()` is transactional - ends current session, validates no unconsumed messages, creates new session for next operations
16. **Session Lifecycle**: Session created at initialization, persists across operations, ends on Wait(), new session created for next transaction

# Socket Package Specification

## Architecture

The socket package implements a TCP-based message bus with client-server architecture supporting request-response patterns within logical channels.

## Components

### Server
- Listens on TCP port with SO_REUSEADDR and optimized buffers (2MB send/recv)
- Maintains a registry of connected clients grouped by channel
- Routes frames between clients based on flags
- Handles graceful shutdown with connection draining
- Each connection handler runs in dedicated goroutine
- flagReceive handlers run in separate goroutines with polling loop (10ms sleep)

### Client
- Establishes TCP connection with TCP_NODELAY and optimized buffers (2MB send/recv)
- Two primary operations:
  - **Send(r io.Reader, timeout)**: Send request, receive response
  - **Receive(incoming, outgoing chan io.Reader, timeout)**: Receive request, send response
- Manages request/response lifecycle with unique request IDs
- Handles chunked payload transmission for large messages (1MB chunks)
- Connection context manages separate reader/writer goroutines

### Registry
- Maps clientID → clientEntry for direct client lookup
- Maps channelID → set of clientEntry for channel-based routing
- Each clientEntry maintains two queues:
  - requestQueue: incoming requests from peers (buffered 1024)
  - responseQueue: outgoing responses to this client (buffered 1024)
- Background goroutine per client drains responseQueue to connection

## Frame Protocol

### Frame Types
- errorFrame (0x03): Error notification
- startFrame (0x04): Begin request/response sequence
- chunkFrame (0x05): Payload data chunk
- endFrame (0x06): Complete request/response sequence

### Frame Flags
- flagRequest (0x01): Client sending request
- flagReceive (0x04): Client polling for requests
- flagResponse (0x08): Client sending response

### Frame Header (68 bytes)
- Version (1 byte)
- Type (1 byte)
- Flags (2 bytes)
- RequestID (16 bytes UUID)
- ClientID (16 bytes UUID)
- ChannelID (16 bytes UUID)
- ClientTimeoutMs (8 bytes)
- Sequence (4 bytes)
- PayloadLength (4 bytes)

### Frame Fields
- sequence (uint32): Chunk sequence number
  - High bit (0x80000000) set indicates final chunk
  - Lower 31 bits contain chunk index (0-based)
  - Transmitted in frame header bytes 60-63
- Payload ([]byte): Frame payload data (max 1MB)

## Request Flow

### Send() Operation (Client1 Perspective)
1. Sends startFrame with flagRequest
2. Waits for startFrame acknowledgment (from server via responseQueue)
3. Sends chunkFrame(s) with payload data
4. Waits for response chunkFrame(s) from Client2
5. Waits for endFrame from Client2 (with flagResponse)
6. Returns assembled response

### Receive() Operation (Client2 Perspective)
1. Sends startFrame with flagReceive (triggers server dequeue)
2. Waits for startFrame from Client1 (dequeued by server)
3. Sends startFrame with flagResponse (acknowledges to Client1)
4. Sends chunkFrame with flagReceive (triggers server dequeue)
5. Receives chunkFrame(s) from Client1 (dequeued by server)
6. Sends response chunkFrame(s) with flagResponse
7. Sends endFrame with flagReceive (triggers server dequeue for Client1's endFrame)
8. Waits for endFrame from Client1 (dequeued by server)
9. Sends endFrame with flagResponse (acknowledges to Client1)

## Server Routing Logic

### Request Frame (flagRequest)
- Enqueue to all peers' requestQueues in same channel
- If no peers exist: enqueue to sender's own requestQueue
- Server does NOT filter by frame type - all frames routed uniformly

### Receive Frame (flagReceive)
- Spawns goroutine that polls in loop:
  - First attempt: dequeue from client's own requestQueue
  - If own queue empty: iterate through all peers' requestQueues
  - If frame found: enqueue to client's responseQueue and exit
  - If nothing available: sleep 10ms and retry
- No blocking - server continuously polls until frame available

### Response Frame (flagResponse)
- Extract ClientID from frame header (target client)
- Enqueue to target client's responseQueue
- Background processResponses goroutine drains responseQueue to connection

## Ordering Guarantees

- Request frames are FIFO per client's requestQueue
- Response frames are FIFO per client's responseQueue
- Queues are isolated per client; no cross-client interference
- Frame transmission order matches enqueue order

## Connection Management

### Client Registration
- First frame from connection establishes ClientID and ChannelID
- Registry creates clientEntry with two buffered queues (1024 each)
- Spawns processResponses goroutine to drain responseQueue
- Adds entry to both clients map and channels map

### Client Unregistration
- Triggered by connection close or context cancellation
- Closes both queues (request, response)
- Removes from clients map (by clientID)
- Removes from channels map (by channelID → clientID)
- Cleans up channel map entry if last client in channel
- Connection context cleanup:
  - Closes writes and errors channels
  - Waits for reader/writer goroutines to exit
  - Closes underlying TCP connection

### Connection Context (connctx)
- Manages TCP connection with separate reader/writer goroutines
- Three channels:
  - writes: normal frames (buffered 1024)
  - reads: incoming frames (buffered 1024)
  - errors: priority error frames (buffered 512)
  - closed: signals shutdown
- Writer prioritizes error frames over normal frames
- Reader sets 2-minute read deadline per frame
- Writer sets 5-second write deadline per frame

## Timeouts

- Frame read timeout: 2 minutes (defaultFrameReadTimeout)
- Frame write timeout: 5 seconds (defaultFrameWriteTimeout)
- Client timeout: configurable per request (default 5 seconds)
- ClientTimeoutMs field in frame header (default 30000ms)
- Timeout of 0 uses defaultTimeout (5 seconds)

## Error Handling

- Error frames (type 0x03) sent on priority errors channel
- Connection errors trigger cleanup and unregistration
- Frame validation errors:
  - errInvalidFrame: wrong frame type received
  - errFrame: error frame received
  - errClosed: client/connection closed
  - errTimeout: operation timeout
- Frame size limits:
  - Max frame payload: 1MB (maxFrameSize)
  - Frame header: 68 bytes fixed
- Protocol version check: only version1 (0x01) supported

## Implementation Details

### Frame Matching
- Client methods filter frames by RequestID
- receiveStartFrame, receiveChunkFrame, receiveEndFrame:
  - Match on specific RequestID OR uuid.Nil (any request)
  - Discard non-matching frames (putFrame and continue)
  - Return errFrame if errorFrame received
  - Return errInvalidFrame if wrong frame type

### Chunk Assembly
- sendDisassembledChunkFrames:
  - Reads from io.Reader in maxFrameSize chunks
  - Sends each chunk with sequence field (index + isFinal flag)
  - Final chunk indicated by high bit in sequence field
- receiveAssembledChunkFrames:
  - Accumulates chunks into bytes.Buffer
  - For flagReceive: sends empty chunkFrame before each receive to trigger dequeue
  - Stops when isFinal() returns true

### Buffer Pooling
- Frame pool: reuses frame structs (sync.Pool)
- Buffer pool: 1MB buffers for chunk reading (sync.Pool)
- Write buffer pool: 8KB+header buffers for small writes (sync.Pool)
- Frames returned to pool via putFrame after processing
- Buffers returned via putBuffer/putWriteBuffer

### Concurrency
- Server: one goroutine per connection in handle()
- Server: one goroutine per flagReceive frame (polling loop)
- Client: two goroutines per connection (reader, writer)
- Registry: one processResponses goroutine per clientEntry
- All frame channels are buffered to reduce blocking
- Mutex protection on registry maps (RWMutex)

## Key Implementation Notes

1. **No Pending Receive Queue**: The original design included a pendingReceive queue, but the final implementation uses polling loops instead
2. **Frame Ownership**: Frames are NOT recycled (putFrame) when enqueued to requestQueue or responseQueue - ownership transfers to the queue
3. **Sequence Field Transmission**: The sequence field is transmitted in the frame header (bytes 60-63) to preserve isFinal flag across the wire
4. **Frame Type Agnostic Server**: Server does not filter frames by type - all flagRequest frames are routed to peers, all flagReceive frames trigger dequeue polling
5. **Polling Sleep**: flagReceive handlers sleep 10ms between dequeue attempts to avoid busy-waiting

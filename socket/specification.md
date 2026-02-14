# Socket Package Specification

## Architecture

The socket package implements a TCP-based message bus with client-server architecture supporting request-response patterns within logical channels.

## Components

### Server
- Listens on TCP port with SO_REUSEADDR and optimized buffers (2MB send/recv)
- Maintains a registry of connected clients grouped by channel
- Routes frames between clients based on flags and channel membership
- Handles graceful shutdown with connection draining
- Each connection handler runs in dedicated goroutine

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
- Each clientEntry maintains three queues:
  - requestQueue: incoming requests from peers (buffered 1024)
  - responseQueue: outgoing responses to this client (buffered 1024)
  - pendingReceive: pending Receive() calls awaiting requests (buffered 1024)
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

### Frame Header (64 bytes)
- Version (1 byte)
- Type (1 byte)
- Flags (2 bytes)
- RequestID (16 bytes UUID)
- ClientID (16 bytes UUID)
- ChannelID (16 bytes UUID)
- ClientTimeoutMs (8 bytes)
- PayloadLength (4 bytes)

### Frame Fields (not in wire format)
- sequence (uint32): Chunk sequence number
  - High bit (0x80000000) set indicates final chunk
  - Lower 31 bits contain chunk index (0-based)
  - Example: 0x80000005 = final chunk, index 5
  - Example: 0x00000003 = chunk 3, more chunks follow
- Payload ([]byte): Frame payload data (max 1MB)

## Request Flow

### Send() Operation (Client Perspective)
1. Client generates unique RequestID
2. Sends startFrame with flagRequest
3. Waits for startFrame acknowledgment (blocks on receiveStartFrame)
4. Sends chunkFrame(s) with payload data via sendDisassembledChunkFrames
   - Reads from io.Reader in maxFrameSize chunks
   - Each chunk sent as separate chunkFrame
5. Waits for response chunkFrame(s) via receiveAssembledChunkFrames
   - Assembles chunks into bytes.Buffer
6. Sends endFrame with flagRequest
7. Waits for endFrame acknowledgment
8. Returns assembled response as io.Reader

### Receive() Operation (Client Perspective)
1. Client sends startFrame with flagReceive and unique origReqID
2. Waits for startFrame from server (blocks on receiveStartFrame with uuid.Nil)
   - Server provides actual RequestID and ClientID of incoming request
3. Client sends startFrame with flagResponse as acknowledgment
4. Client receives chunkFrame(s) via receiveAssembledChunkFrames
   - Uses flagReceive to signal server to send next chunk
   - Assembles request payload into bytes.Buffer
5. Client sends assembled request to incoming channel
6. Client waits for response from outgoing channel
7. Client sends response chunkFrame(s) with flagResponse
8. Client sends endFrame with flagReceive
9. Client waits for endFrame acknowledgment

## Server Routing Logic

### Request Frame (flagRequest)
- Get all peers in same channel (excluding sender)
- If no peers exist: enqueue to sender's own requestQueue
- If peers exist: enqueue to ALL peers' requestQueues
- Frames are always enqueued to support late connections
- If a peer calls Receive() before Send(), frames wait in queue
- If a peer calls Receive() after Send(), frames are already queued

### Receive Frame (flagReceive)
- First attempt: dequeue from client's own requestQueue
- If own queue empty: iterate through all peers' requestQueues
- If nothing available: enqueue to client's pendingReceive queue (unused currently)
- When frame found: write directly to client's connection
- Receive blocks until a request frame is available

### Response Frame (flagResponse)
- Extract ClientID from frame header (target client)
- Only chunkFrames with flagResponse are routed
- startFrame and endFrame with flagResponse are acknowledgments (discarded)
- Enqueue chunkFrame to target client's responseQueue
- Background processResponses goroutine drains responseQueue:
  - Blocking send to connctx.writes channel
  - Ensures FIFO delivery of responses

## Ordering Guarantees

- Request frames are FIFO per client's requestQueue
- Response frames are FIFO per client's responseQueue
- Queues are isolated per client; no cross-client interference
- Frame transmission order matches enqueue order

## Connection Management

### Client Registration
- First frame from connection establishes ClientID and ChannelID
- Registry creates clientEntry with three buffered queues (1024 each)
- Spawns processResponses goroutine to drain responseQueue
- Adds entry to both clients map and channels map

### Client Unregistration
- Triggered by connection close or context cancellation
- Closes all three queues (request, response, pendingReceive)
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
  - Frame header: 64 bytes fixed
- Protocol version check: only version1 (0x01) supported

## Late Connection Handling

### Send Before Receive
- Client1 calls Send() and sends request frames
- Server enqueues all frames to Client2's requestQueue
- Client2 later calls Receive()
- Server dequeues frames from Client2's requestQueue
- Frames are delivered in order

### Receive Before Send
- Client2 calls Receive() and sends flagReceive frame
- Server finds no frames in any requestQueue
- Server enqueues flagReceive to Client2's pendingReceive queue (currently unused)
- Client1 later calls Send()
- Server enqueues request frames to Client2's requestQueue
- Client2's Receive continues polling and dequeues frames

### Current Implementation
- Request frames are always enqueued to requestQueues
- Receive operations poll requestQueues until frames available
- No direct frame delivery from pending receives
- This ensures all frames in a sequence are handled consistently

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
  - Sends each chunk with index and length fields
  - Length field indicates total expected chunks (currently 0)
- receiveAssembledChunkFrames:
  - Accumulates chunks into bytes.Buffer
  - Stops when length == 0 or length == (index + 1)
  - For flagReceive: sends chunk request before each receive

### Buffer Pooling
- Frame pool: reuses frame structs (sync.Pool)
- Buffer pool: 1MB buffers for chunk reading (sync.Pool)
- Write buffer pool: 8KB+header buffers for small writes (sync.Pool)
- Frames returned to pool via putFrame after processing
- Buffers returned via putBuffer/putWriteBuffer

### Concurrency
- Server: one goroutine per connection in handle()
- Client: two goroutines per connection (reader, writer)
- Registry: one processResponses goroutine per clientEntry
- All frame channels are buffered to reduce blocking
- Mutex protection on registry maps (RWMutex)

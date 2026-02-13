# Socket Package Context

## Architecture Overview

The socket package implements a server-side connected client registry that manages multiple concurrent logical clients with strict FIFO ordering per client.

## Core Components

### Registry (registry.go)
- Manages all connected clients
- Each client has two independent queues:
  - **Request Queue**: Contains request frames from other clients in the same channel
  - **Response Queue**: Contains response frames to be sent to that specific client
- Queues enforce strict FIFO ordering, isolated per client

### Server (server.go)
- Accepts TCP connections on localhost
- Routes frames based on flags:
  - **flagRequest (0x01)**: Enqueues to peers' request queues (or own if no peers)
  - **flagReceive (0x04)**: Dequeues from own request queue first, then peers' queues, sends directly to client
  - **flagResponse (0x08)**: Routes to original requester's response queue
  - Unknown flags: Discarded

### Client (client.go)
- Three main operations:
  - **Send()**: Sends request frames with flagRequest to server
  - **Receive()**: Sends frame with flagReceive, waits for dequeued request
  - **Respond()**: Sends response frames with flagResponse back to requester

## Frame Structure
- Version (byte)
- Type (byte): startFrame, chunkFrame, endFrame, errorFrame
- Flags (uint16): flagRequest, flagReceive, flagResponse
- RequestID (UUID)
- ClientID (UUID)
- ChannelID (UUID)
- ClientTimeoutMs (uint64)
- Payload ([]byte)

## Frame Routing Logic

### When Client Calls Send()
1. Client sends frames with flagRequest
2. Server enqueues to all other clients' request queues in same channel
3. If no other clients exist, enqueues to sender's own request queue

### When Client Calls Receive()
1. Client sends frame with flagReceive
2. Server dequeues from client's own request queue first
3. If empty, dequeues from other clients' request queues in same channel
4. Server sends dequeued frame directly to client (bypasses response queue)
5. If no requests available, sends errorFrame

### When Client Calls Respond()
1. Client sends frames with flagResponse
2. Server routes to original requester's response queue
3. Response queue processor sends to client in FIFO order

## Key Design Decisions

1. **No Handshake**: Removed handshake mechanism for simplicity
2. **No PeerClientID**: Frames don't track peer relationships
3. **Direct Send on Receive**: Receive() bypasses response queue for immediate delivery
4. **Flag-Based Routing**: Server uses flags to determine frame routing
5. **Per-Client Isolation**: Request/response queues are completely isolated per client

## Files Structure
- `server.go`: Server implementation with frame routing logic
- `client.go`: Client implementation with Send/Receive/Respond methods
- `registry.go`: Client registry with per-client queues
- `connctx.go`: Connection context with frame read/write logic
- `shared.go`: Shared constants and error definitions
- `bufferpool.go`: Buffer pooling for performance
- `specification.md`: Detailed specification document

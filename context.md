# PulsyFlux Context

## Project Overview
A message bus system that is reliable and flexible in message handling, written in Go.

## Architecture Summary

### Frame Structure (64-byte header)
- **Version** (1 byte): Protocol version
- **Type** (1 byte): Frame type (errorFrame=0x03, startFrame=0x04, chunkFrame=0x05, endFrame=0x06)
- **Flags** (2 bytes): Reserved for future use
- **RequestID** (16 bytes): UUID for request tracking
- **ClientID** (16 bytes): UUID of sender
- **PeerClientID** (16 bytes): UUID of intended recipient
- **ClientTimeoutMs** (8 bytes): Timeout in milliseconds
- **PayloadLen** (4 bytes): Length of payload data

### Frame Types (Only 4 exist)
1. **errorFrame (0x03)**: Created by server, sent to clients for error conditions
2. **startFrame (0x04)**: Initiates a request/response exchange
3. **chunkFrame (0x05)**: Carries payload data
4. **endFrame (0x06)**: Terminates a request/response exchange

### Frame Exchange Pattern
All communication follows: **send start → send chunks → send end → receive assembled response**
- No interleaved waiting between send operations
- Payload belongs in chunks and endFrame, not startFrame
- Response follows same pattern: start (for peerClientID) → chunks (for payload) → end (for completion)

### Client Roles
- **roleConsumer (0x01)**: Client requesting services
- **roleProvider (0x02)**: Client providing services
- Roles are socket-layer concepts only; higher layers handle business logic

## Key Components

### socket/frame.go
- 40-byte header structure (legacy, being phased out)
- ClientID at bytes 20-36, PayloadLen at 36-40
- Fixed write method to match read method layout

### socket/connctx.go
- 64-byte header frame structure (current)
- Contains readFrame/writeFrame methods
- Frame pooling via getFrame/putFrame
- **CRITICAL**: bufferPool provides temporary buffers; putFrame() clears payload

### socket/server.go
- Uses requestHandler for start/chunk/end frames (timeout tracking)
- Routes other frames directly to peers via PeerClientID
- Refreshes timeout context in endFrame before forwarding
- **CRITICAL**: Must panic on nil request (no nil checks) - indicates architecture issue

### socket/client.go (Unified Client)
- Replaces separate Provider/Consumer implementations
- Constructor: `NewClient(addr, channelID, role)`
- **Handshake**: Bidirectional exchange using start/chunk/end frames
  - Initiator: Uses `exchange()` to send (start→chunks→end) and `receiveAssembledResponse()` to wait for peer's response
  - Responder: Uses `Receive()` to get handshake request, then `Send()` to respond
  - Validates same channelID and different roles
  - Sets peerID after successful validation
- **exchange()**: Common function for both handshake and Send
  - Sends start → chunks → end
  - Calls receiveAssembledResponse() to receive and assemble response
- **receiveAssembledResponse()**: Assembles response frames for a specific requestID
  - Uses temporary buffer to accumulate chunk payloads
  - Returns assembled payload and peerClientID
- **Send()**: Sends request using exchange(), returns assembled response as io.Reader
- **Receive()**: Receives incoming requests from peer
  - Uses streamReqs map to assemble multiple concurrent requests
  - Returns requestID and assembled payload when endFrame arrives
  - Used by responder to receive handshake and normal requests

### socket/peers.go
- Simple registry mapping clientID to connctx
- Methods: get, set, delete

### socket/reqworker.go
- Forwards frames to peer using peers.get(peerClientID)

### socket/shared.go
- Error variables: errTimeout, errPeerError, errClosed

## Design Principles

1. **No Response Frame Types**: Only 4 frame types exist; responses use same frame types
2. **Server as Router**: Server routes frames between peers, doesn't create frames (except errors)
3. **Peer-to-Peer**: Socket layer is pure P2P routing; no Consumer/Provider distinction at this level
4. **Frame Flow Consistency**: All exchanges follow start → chunks → end pattern
5. **Timeout Management**: Server tracks timeouts via requestHandler for start/chunk/end frames
6. **Error Handling**: Server creates error frames; clients handle them
7. **Frame Pooling**: Frames must be processed before returning to pool (payload cleared on putFrame)

## Recent Changes

### Client Unification
- Removed separate Provider and Consumer types
- Created unified Client with role parameter
- **Handshake removed from socket layer** - needs to be implemented at application layer or server needs channel-based routing

### Frame Exchange Simplification
- Removed premature frame consumption (waitForStartFrame, waitForChunks)
- Simplified exchange() to send all frames before waiting for response
- receiveAssembledResponse() properly accumulates chunks and assembles final payload

### Payload Assembly
- startFrame captures peerClientID
- chunkFrames accumulate into buffer
- endFrame triggers final assembly and return

### Known Issues
- **Handshake requires server support**: Handshake uses flagRegistration but server needs to route these frames by channelID instead of PeerClientID
- **Workaround**: Application layer must handle peer discovery:
  1. Both clients connect to server
  2. Use out-of-band mechanism to exchange clientIDs
  3. Manually set peerID on each client
  4. Then use normal Send/Receive
- **Alternative**: Implement server-side channel registry to route registration frames (requires server changes)

## Critical Notes

- **Frame Lifecycle**: Frames from pool must be processed before putFrame() call
- **Nil Checks**: Server must panic on nil request - indicates critical bug
- **Handshake Required**: Client must complete handshake before Send/Receive
- **Role Validation**: Peers must have same channelID but different roles
- **Timeout Refresh**: Server refreshes context timeout on endFrame

## File Locations
- Frame definitions: `e:\github\pulsyflux\socket\frame.go`, `e:\github\pulsyflux\socket\connctx.go`
- Server: `e:\github\pulsyflux\socket\server.go`
- Client: `e:\github\pulsyflux\socket\client.go`
- Peers registry: `e:\github\pulsyflux\socket\peers.go`
- Request worker: `e:\github\pulsyflux\socket\reqworker.go`
- Shared types: `e:\github\pulsyflux\socket\shared.go`

## Next Steps
- Test unified Client implementation
- Implement higher-level message bus on top of socket layer
- Add comprehensive error handling and recovery
- Performance testing and optimization


## Current Implementation Status

### Handshake Implementation
- Handshake has been removed from the socket layer as it requires server-side channel registry
- Clients automatically send a discovery startFrame on connection
- Clients capture peerID from the first startFrame they receive
- **LIMITATION**: Server routes frames by PeerClientID, so clients cannot discover each other without knowing clientIDs in advance

### Test Status
- Tests are currently failing because they expect automatic peer discovery
- This functionality requires server changes to support channel-based routing or broadcast
- **SOLUTION NEEDED**: Either modify server to support channel registry, or update tests to manually exchange clientIDs

### Recommended Fix
Option 1: Add channel registry to server (requires server.go changes - not allowed)
Option 2: Add SetPeerID() method and update tests to manually set peer IDs after both clients connect
Option 3: Use out-of-band mechanism (e.g., external service) for peer discovery

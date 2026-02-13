# Socket Package Refactoring Summary

## Changes Made

### Core Architecture
- Removed handshake mechanism entirely
- Implemented per-client request and response queues with strict FIFO ordering
- Created new `registry.go` to manage connected clients with isolated queues

### New Files
- **registry.go**: Client registry with per-client request/response queues
  - `clientEntry`: Holds client state with dedicated request and response queues
  - `register()`: Registers client and starts response processor
  - `enqueueRequest()`: Adds request frame to client's request queue
  - `dequeueRequest()`: Removes request frame from client's request queue
  - `enqueueResponse()`: Adds response frame to client's response queue
  - `processResponses()`: Goroutine that transmits responses in FIFO order

### Modified Files

#### server.go
- Replaced `peers` and `requestHandler` with `registry`
- Simplified `handle()` to route frames to appropriate queues
- Request frames (start/chunk/end) → client's request queue
- Response frames → client's response queue
- Removed all handshake logic

#### client.go
- Removed `handshakeClient` embedding
- Added explicit `clientID`, `peerID`, `channelID` fields
- Removed `doHandshake()` calls from `Send()` and `Respond()`
- Added `Receive()` method to dequeue request frames
- Simplified `respondStartFrame()` to handle first request

#### connctx.go
- Removed `flagHandshake` and `flagPeerNotAvailable` constants
- Kept only `flagNone` and `flagBroadcast`

#### shared.go
- Removed `request` struct (no longer needed)
- Removed `workerQueueTimeout` constant
- Kept only essential error types

### Files to Remove
The following files are no longer needed and can be deleted:
- handshake.go
- mapper.go
- peers.go
- reqhandler.go
- reqworker.go

## Specification Compliance

✓ Server-side client registry manages multiple concurrent clients
✓ Each client has independent request and response queues
✓ Request queue contains frames from OTHER clients in same channel
✓ Strict FIFO ordering enforced per queue
✓ Send() enqueues request frames to all other clients' request queues (or own if alone)
✓ Receive() dequeues from client's request queue
✓ Respond() enqueues to originating client's response queue
✓ Response frames transmitted in exact enqueue order
✓ No frame intermixing between clients

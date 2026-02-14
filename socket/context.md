# Socket Package Context

## Current State

The socket package implements a message bus with client-server architecture for reliable message handling between peers on the same channel.

## Architecture

### Client
- Single connection per client
- Operations: Send() and Receive()
- Mutex-protected to prevent concurrent operations on same client
- Frame-based protocol with start/chunk/end frames

### Server
- Registry-based routing using channelID
- Routes frames between clients on same channel
- Handles flagRequest, flagReceive, and flagResponse

### Frame Protocol
- Header: 68 bytes (version, type, flags, IDs, timeout, sequence, payload length)
- Types: startFrame, chunkFrame, endFrame, errorFrame
- Flags: flagRequest, flagReceive, flagResponse
- Sequence field includes final flag (bit 31)

## Edge Case Testing

Created comprehensive tests without sleep/wait calls:

### Passing Tests
1. **Connection refused** - Handles failed connections
2. **Send after close** - Returns errClosed
3. **Multiple close** - Idempotent close
4. **Empty payload** - Sends at least one chunk frame
5. **Large payload** - 5MB transfers work
6. **Zero timeout** - Uses default timeout
7. **Read errors** - Propagates io.Reader errors
8. **Concurrent operations** - Returns errOperationInProgress
9. **Sequential requests** - Multiple request/response cycles
10. **Boundary sizes** - 1, 1KB, 8KB, maxFrameSize payloads

### Known Issues

#### Critical: Frame Ordering Violation
**Test**: TestClientMultipleConcurrent  
**Symptom**: "invalid frame" errors when multiple clients send/receive concurrently  
**Root Cause**: Frames from different requests are interleaving, violating protocol order (start→chunk→end)  
**Impact**: Protocol breaks down under concurrent load  
**Status**: UNRESOLVED - This is a serious infrastructure issue

## Client Improvements Made

1. **Mutex Protection**: Added opMu to prevent concurrent Send/Receive on same client
2. **Close Safety**: 
   - Close ctx.closed before conn.Close() to signal goroutines
   - Wait for goroutines before closing channels
   - Idempotent close with done channel check
3. **Empty Payload**: Always send at least one chunk frame (even if empty)
4. **Error Clarity**: Added errOperationInProgress for concurrent operation attempts
5. **Channel Checks**: All send/receive operations check ctx.closed

## Frame Routing Rules

### flagRequest frames
- startFrame: Acknowledge back to sender (not routed)
- chunkFrame: Route to peers
- endFrame: Acknowledge back to sender (not routed)

### flagReceive frames
- All types: Poll queues and deliver available frames

### flagResponse frames
- All types: Route to target client via responseQueue

## Test Strategy

All tests use proper synchronization:
- Channels for coordination
- sync.WaitGroup for completion
- No time.Sleep() or arbitrary waits
- Tests expose real infrastructure issues

## Outstanding Work

1. **Fix frame interleaving** - Multiple concurrent clients cause invalid frame errors
2. **Investigate server routing** - Why do frames from different requests mix?
3. **Add frame sequence validation** - Detect out-of-order frames earlier
4. **Consider per-request channels** - Isolate frame streams by requestID

# Socket Package Issues

## Issue #1: Frame Interleaving with Multiple Concurrent Receivers

**Status**: IDENTIFIED - NOT FIXED

**Test**: TestClientMultipleConcurrent

**Symptom**: "invalid frame" errors when multiple clients send/receive concurrently

### Investigation Log

Added logging to track frame flow:
- Client.Send() logs START/COMPLETE with reqID
- Client.Receive() logs START with origReqID and which request it receives
- receiveStartFrame() logs every frame received with type and reqID

### Key Findings from Logs

```
[Client 31ad578d] Receive: START origReqID=fe20f6dc
[Client 31ad578d] receiveStartFrame: got frame type=4 reqID=95e29baf (expecting reqID=00000000)
[Client 31ad578d] Receive: Got request reqID=95e29baf from client=cbb0c5fc

[Client 8e6bc9b5] Receive: START origReqID=30917783
[Client 8e6bc9b5] receiveStartFrame: got frame type=4 reqID=feed0a1e (expecting reqID=00000000)
[Client 8e6bc9b5] Receive: Got request reqID=feed0a1e from client=8d447ff6
```

Then errors:
```
Receiver 0 error: invalid frame
Receiver 1 error: invalid frame
Receiver 2 error: invalid frame
```

### Root Cause

**Multiple receivers are receiving frames from the SAME requests.**

When multiple clients call Receive() on the same channel:
1. Client 31ad578d receives request 95e29baf
2. Client 8e6bc9b5 receives request feed0a1e
3. But then BOTH clients start receiving frames from BOTH requests
4. This causes "invalid frame" errors because they receive chunk/`end frames from requests they didn't initiate

**The server is broadcasting frames to ALL receivers on a channel instead of routing each request to exactly ONE receiver.**

### Expected Behavior

Each request should be assigned to exactly ONE receiver:
- Request 95e29baf → Client 31ad578d (and ONLY this client)
- Request feed0a1e → Client 8e6bc9b5 (and ONLY this client)
- Request dfc1c1d8 → Client 1c1178b2 (and ONLY this client)

### Server Routing Problem

The server needs to:
1. When a receiver calls Receive(), assign it to handle the NEXT available request
2. Route ALL frames (start/chunk/end) for that request ONLY to that specific receiver
3. Not broadcast frames to other receivers on the channel

### Next Steps

1. Review server.go routing logic for flagRequest frames
2. Check how frames are enqueued to peer queues
3. Ensure each request is assigned to exactly one receiver
4. Add request-to-receiver mapping to prevent frame interleaving

---

## Proposed Solution: Request Assignment Tracking with Shared Channel Queue

**Status**: PROPOSED

### Problem Summary

Current behavior broadcasts ALL flagRequest frames to ALL receivers:
```go
for _, peer := range peers {
    peer.enqueueRequest(f)  // Broadcasts to ALL
}
```

This causes frame interleaving:
- Receiver A gets startFrame for request X
- Receiver B gets startFrame for request Y
- Both receivers get chunkFrames from BOTH requests
- Result: "invalid frame" errors

**Additional problem**: Current flagReceive implementation spawns a polling goroutine:
```go
go func() {
    for {
        if req, ok := entry.dequeueRequest(); ok { ... }
        time.Sleep(10 * time.Millisecond)  // Wasteful polling!
    }
}()
```
With N receivers, this creates N goroutines polling every 10ms = 10N polls/second doing nothing.

### Solution: Shared Channel Queue + Request Assignment

**Core Ideas**:
1. Use ONE shared queue per channel for startFrames (eliminates polling)
2. Receivers block on shared queue (efficient, no busy-waiting)
3. Once receiver dequeues startFrame, assign entire request to that receiver
4. Route subsequent frames (chunk/end) directly to assigned receiver

**Implementation**:

1. **Add to registry**:
   - `channelQueues map[uuid.UUID]chan *frame` - shared queue per channel for startFrames
   - `requestAssignments map[uuid.UUID]uuid.UUID` - tracks requestID → receiverClientID

2. **When sender sends startFrame (flagRequest)**:
   - Enqueue to shared channel queue (not to individual peer queues)
   - All receivers waiting on that channel queue compete for it
   - First available receiver gets it (natural load balancing)

3. **When receiver polls (flagReceive)**:
   - Spawn goroutine that BLOCKS on shared channel queue (no polling!)
   - When frame received: `assignRequest(requestID, receiverClientID)`
   - Send frame to receiver via responseQueue

4. **When sender sends chunk/endFrame (flagRequest)**:
   - Check if requestID is assigned
   - If assigned: Route ONLY to assigned receiver's responseQueue
   - If not assigned: Error condition (shouldn't happen)

5. **When request completes**:
   - On endFrame with flagResponse: `unassignRequest(requestID)`

### Detailed Frame Flow

#### Scenario: 3 Senders, 2 Receivers on same channel

```
Sender1          Sender2          Server                 Receiver1        Receiver2
  |                |                  |                        |               |
  |                |                  | <--flagReceive---------+               |
  |                |                  |   spawn goroutine      |               |
  |                |                  |   BLOCKS on channelQ   |               |
  |                |                  |                        |               |
  |                |                  | <--flagReceive--------------------+   |
  |                |                  |   spawn goroutine                 |   |
  |                |                  |   BLOCKS on channelQ              |   |
  |                |                  |                        |           |   |
  +--startFrame--->|                  |                        |           |   |
  |  [reqA]        |                  |                        |           |   |
  |  [flagRequest] |                  |                        |           |   |
  |                |                  +--enqueue to channelQ   |           |   |
  |                |                  |                        |           |   |
  |                |                  +--goroutine1 unblocks-->|           |   |
  |                |                  |  ASSIGN reqA->Rcv1     |           |   |
  |                |                  +--send via responseQ--->|           |   |
  |                |                  |                        |           |   |
  |                +--startFrame----->|                        |           |   |
  |                |  [reqB]          |                        |           |   |
  |                |  [flagRequest]   |                        |           |   |
  |                |                  +--enqueue to channelQ   |           |   |
  |                |                  |                        |           |   |
  |                |                  +--goroutine2 unblocks-------------->|   |
  |                |                  |  ASSIGN reqB->Rcv2                |   |
  |                |                  +--send via responseQ--------------->|   |
  |                |                  |                        |           |   |
  +--chunkFrame--->|                  |                        |           |   |
  |  [reqA]        |                  |                        |           |   |
  |  [flagRequest] |                  |                        |           |   |
  |                |                  +--check assignment      |           |   |
  |                |                  |  reqA->Rcv1            |           |   |
  |                |                  +--route to Rcv1 ONLY--->|           |   |
  |                |                  |  (NOT to Rcv2)         |           |   |
  |                |                  |                        |           |   |
  |                +--chunkFrame----->|                        |           |   |
  |                |  [reqB]          |                        |           |   |
  |                |  [flagRequest]   |                        |           |   |
  |                |                  +--check assignment      |           |   |
  |                |                  |  reqB->Rcv2                        |   |
  |                |                  +--route to Rcv2 ONLY--------------->|   |
  |                |                  |  (NOT to Rcv1)         |           |   |
  |                |                  |                        |           |   |
  +--endFrame----->|                  |                        |           |   |
  |  [reqA]        |                  |                        |           |   |
  |  [flagRequest] |                  |                        |           |   |
  |                |                  +--check assignment      |           |   |
  |                |                  |  reqA->Rcv1            |           |   |
  |                |                  +--route to Rcv1-------->|           |   |
  |                |                  |                        |           |   |
  |                |                  |                        +--endFrame-+   |
  |                |                  | <----------------------+  [reqA]   |   |
  |                |                  |  [flagResponse]        |           |   |
  |                |                  +--UNASSIGN reqA         |           |   |
  |                |                  +--route to Sender1----->|           |   |
  | <--endFrame----+                  |                        |           |   |
  |  [reqA]        |                  |                        |           |   |
```

### Key Changes to Frame Routing

**BEFORE (Current - Broken)**:
```go
// All flagRequest frames broadcast to ALL peers
if f.Flags&flagRequest != 0 {
    peers := s.registry.getChannelPeers(entry.channelID, clientID)
    for _, peer := range peers {
        peer.enqueueRequest(f)  // BROADCAST - causes interleaving
    }
}

// flagReceive spawns polling goroutine
if f.Flags&flagReceive != 0 {
    go func() {
        for {
            if req, ok := entry.dequeueRequest(); ok { ... }
            time.Sleep(10 * time.Millisecond)  // WASTEFUL POLLING
        }
    }()
}
```

**AFTER (Proposed - Fixed)**:
```go
// startFrame goes to shared channel queue
if f.Flags&flagRequest != 0 && f.Type == startFrame {
    channelQueue := s.registry.getChannelQueue(entry.channelID)
    channelQueue <- f  // Single queue, no broadcast
}

// chunk/endFrame goes to assigned receiver only
if f.Flags&flagRequest != 0 && (f.Type == chunkFrame || f.Type == endFrame) {
    if receiverID, ok := s.registry.getAssignedReceiver(f.RequestID); ok {
        if receiver, ok := s.registry.get(receiverID); ok {
            receiver.enqueueResponse(f)  // Direct routing, no broadcast
        }
    }
}

// flagReceive blocks on shared queue (no polling!)
if f.Flags&flagReceive != 0 {
    go func() {
        channelQueue := s.registry.getChannelQueue(entry.channelID)
        req := <-channelQueue  // BLOCKS until request available
        s.registry.assignRequest(req.RequestID, clientID)
        entry.enqueueResponse(req)
    }()
}
```

### Impact on Client-Server Interaction

**Client-side**: NO CHANGES
- Clients continue to send/receive frames exactly as before
- Client code is unaware of server-side routing changes

**Server-side**: Routing logic changes
1. **startFrame routing**: Shared queue instead of broadcast
2. **chunk/endFrame routing**: Direct to assigned receiver instead of broadcast
3. **flagReceive handling**: Blocking instead of polling
4. **Request assignment**: New tracking to maintain request→receiver mapping

**Performance improvements**:
- Eliminates wasteful polling (10N polls/second → 0)
- Reduces memory pressure (fewer queued duplicate frames)
- Natural load balancing (first available receiver gets request)
- Scales efficiently with any number of receivers

**Correctness improvements**:
- Eliminates frame interleaving (each request goes to exactly one receiver)
- Prevents "invalid frame" errors in concurrent scenarios
- Maintains FIFO ordering per request

### Code Changes Required

**SERVER-SIDE ONLY - NO CLIENT CHANGES NEEDED**

**registry.go**:
- Add `channelQueues map[uuid.UUID]chan *frame` - shared queue per channel
- Add `requestAssignments map[uuid.UUID]uuid.UUID` - request to receiver mapping
- Add `getChannelQueue(channelID uuid.UUID) chan *frame` - get or create channel queue
- Add `assignRequest(requestID, receiverClientID uuid.UUID)`
- Add `getAssignedReceiver(requestID uuid.UUID) (uuid.UUID, bool)`
- Add `unassignRequest(requestID uuid.UUID)`
- Cleanup channel queues and assignments on unregister

**server.go**:
- Modify flagRequest handler:
  - If startFrame: enqueue to shared channel queue (not broadcast)
  - If chunk/endFrame: check assignment, route to assigned receiver only
- Modify flagReceive handler:
  - Spawn goroutine that blocks on shared channel queue (no polling loop!)
  - After receiving frame: call assignRequest()
  - Send frame to receiver via responseQueue
- Modify flagResponse handler:
  - On endFrame: call unassignRequest() to cleanup

**client.go**:
- No changes required - clients already filter frames by requestID

### Edge Cases Handled

1. **Multiple concurrent requests**: Each gets its own assignment
2. **Receiver disconnects mid-request**: Assignment cleanup on unregister
3. **Late-arriving frames**: Fallback to broadcast if no assignment
4. **Concurrent map access**: Use existing registry.mu (RWMutex) for thread-safe map operations

### Locking Strategy

**Server-side** (this solution):
- Reuse existing `registry.mu` (RWMutex) for `requestAssignments` map access
- Read lock for checking assignments (frequent)
- Write lock for assign/unassign operations (infrequent)

**Client-side** (already exists, necessary):
- `opMu` on Client (not connctx) prevents concurrent Send()/Receive()
- This is a **business logic constraint**: a client can't be both sender and receiver simultaneously
- Without lock: both operations would read from same `c.ctx.reads` channel and steal each other's frames
- Lock is on Client (not connctx) because:
  - It enforces a logical rule about client roles, not just resource protection
  - connctx is a connection resource; Client is a logical participant
  - The protocol design requires one operation per client at a time

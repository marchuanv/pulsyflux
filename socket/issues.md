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

## Proposed Solution: Request Assignment Tracking

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

### Solution: Request-to-Receiver Assignment

**Core Idea**: Once a receiver dequeues a startFrame, assign that entire request to that receiver only.

**Implementation**:

1. Add `requestAssignments map[uuid.UUID]uuid.UUID` to registry (requestID → receiverClientID)

2. **When receiver polls (flagReceive)**:
   - Dequeue frame from requestQueue
   - If frame is startFrame: `assignRequest(requestID, receiverClientID)`
   - Send frame to receiver via responseQueue

3. **When sender sends frames (flagRequest)**:
   - If startFrame: Broadcast to ALL peers (current behavior)
   - If chunk/endFrame: Check if requestID is assigned
     - If assigned: Route ONLY to assigned receiver
     - If not assigned: Broadcast to ALL peers (fallback)

4. **When request completes**:
   - On endFrame with flagResponse: `unassignRequest(requestID)`

### Message Flow

```
Sender              Server                  Receiver
  |                   |                         |
  |--startFrame------>|                         |
  |  [flagRequest]    |--broadcast to ALL------>|
  |                   |                         |
  |                   |         <--poll---------|
  |                   |  dequeue startFrame     |
  |                   |  *** ASSIGN reqID->rcvID|
  |                   |--send to receiver------>|
  |                   |                         |
  |--chunkFrame------>|                         |
  |  [flagRequest]    |--check assignment------>|
  |                   |  route to assigned ONLY |
  |                   |                         |
  |--endFrame-------->|                         |
  |  [flagRequest]    |--check assignment------>|
  |                   |  route to assigned ONLY |
  |                   |                         |
  |                   |         <--endFrame-----|
  |                   |  [flagResponse]         |
  |                   |  *** UNASSIGN reqID     |
```

### Key Benefits

1. **No starvation**: All receivers can pick up new requests (startFrames are broadcast)
2. **No interleaving**: Once assigned, all frames for a request go to one receiver
3. **Minimal changes**: Only ~20 lines of code added
4. **Backward compatible**: Fallback to broadcast if no assignment exists

### Code Changes Required

**SERVER-SIDE ONLY - NO CLIENT CHANGES NEEDED**

**registry.go**:
- Add `requestAssignments map[uuid.UUID]uuid.UUID`
- Add `assignRequest(requestID, receiverClientID uuid.UUID)`
- Add `getAssignedReceiver(requestID uuid.UUID) (uuid.UUID, bool)`
- Add `unassignRequest(requestID uuid.UUID)`

**server.go**:
- Modify flagRequest handler: check assignment for chunk/endFrame
- Modify flagReceive polling: call assignRequest() after dequeuing startFrame
- Modify flagResponse handler: call unassignRequest() on endFrame

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

# Socket Implementation Context

## Current Status

### Passing Tests
- `TestSendReceiveRespond` - Basic send/receive/respond flow between two clients
- `TestNoOtherClients` - Client sending to itself when no peers exist
- `TestReceiveFromPeerQueue` - Client receiving from peer's queue

### Failing Tests
- `TestMultiplePeers` - Multiple peers receiving same request (needs work)
- `TestConcurrentRequests` - Concurrent request handling (needs work)

## Key Implementation Details

### Frame Routing Logic

1. **START Frames with flagRequest**:
   - Try to send directly to peers with pending receives
   - If no pending receives, enqueue to peer request queues
   - Send ACK to sender when enqueued to own queue (no peers) or successfully enqueued to peers

2. **CHUNK/END Frames with flagRequest**:
   - Send directly to first peer (assumes START already established connection)
   - No pending receive check needed for continuation frames

3. **Response Frames (flagResponse)**:
   - Route directly back to original requester using ClientID from frame
   - Blocking send to ensure delivery

4. **Receive Frames (flagReceive)**:
   - Try to dequeue from own request queue first
   - Then try peers' request queues
   - If no requests available, enqueue as pending receive

### Client Response Flow

When a client calls `Respond()`:
1. Sends a receive frame to signal readiness
2. Waits for START frame from peer
3. Extracts requester's ClientID from incoming frame
4. Uses requester's ClientID in all response frames (START, CHUNK, END)
5. This ensures server routes responses back to original requester

### Critical Fixes Applied

1. **Response Routing**: Changed from using responder's ClientID to using requester's ClientID in response frames
2. **Chunk Frame Responses**: Always send chunk response even if payload is empty to maintain request-response pairing
3. **Direct Peer Routing**: CHUNK/END frames bypass pending receive mechanism and go directly to first peer
4. **ACK for Enqueued Requests**: Send ACK when START frame is enqueued (no immediate responder available)

### Known Issues

1. **Multiple Peers**: When multiple peers exist, only first peer receives CHUNK/END frames. Need to track which peer is handling each request.
2. **Concurrent Requests**: Multiple concurrent requests may interfere with each other's routing.

## Next Steps

To fix remaining tests:
1. Track active request-peer mappings (requestID -> peerID)
2. Route CHUNK/END frames to the specific peer handling that request
3. Handle race conditions in concurrent request scenarios

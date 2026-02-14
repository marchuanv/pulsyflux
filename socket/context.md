# Socket Package Context

## Problem Analysis

The TestSendReceive test is timing out. No server logs appear, meaning the server never receives frames from clients.

## Current Understanding

### Client.Send() Flow
1. Sends startFrame with flagRequest
2. **Waits for startFrame acknowledgment** (receiveStartFrame blocks)
3. Never proceeds because server doesn't acknowledge

### Client.Receive() Flow  
1. Sends startFrame with flagReceive
2. **Waits for startFrame from server** (receiveStartFrame with uuid.Nil blocks)
3. Never proceeds because server doesn't send anything

### Server Behavior
The server only routes frames it receives. It does NOT send acknowledgments for startFrame.

## Root Cause

**The server is missing acknowledgment logic for startFrame.**

According to the specification:
- "Waits for startFrame acknowledgment (blocks on receiveStartFrame)" - Client.Send expects ack
- "Waits for startFrame from server (blocks on receiveStartFrame with uuid.Nil)" - Client.Receive expects server to send startFrame when request available

## Solution Needed

The server must:

1. **For flagRequest + startFrame**: Send acknowledgment back to sender
2. **For flagRequest + endFrame**: Send acknowledgment back to sender  
3. **For flagReceive + startFrame**: 
   - If request available: Send the request's startFrame to receiver
   - If no request: Store in pendingReceive, send later when request arrives
4. **For flagReceive + chunkFrame**: Send next chunk from request
5. **For flagReceive + endFrame**: Send acknowledgment

The key insight: The server acts as a broker that acknowledges and forwards frames between clients.


## Frame Sequence Issue Found

The client's `receiveAssembledChunkFrames` with flagReceive sends a chunkFrame on EVERY iteration to poll for the next chunk. This is the "polling" mechanism.

The server was acknowledging ALL flagRequest frames including chunkFrames, creating an infinite echo loop where:
1. Client sends chunkFrame with flagReceive to request next chunk
2. Server echoes it back as acknowledgment
3. Client receives empty chunk, sends another request
4. Loop continues forever

## Fix

Server should only acknowledge startFrame and endFrame with flagRequest, NOT chunkFrame.
ChunkFrames with flagRequest should be routed to peers, not echoed back.


## Chunk Delivery Issue

From logs:
- Client1 sends: chunk index=0 payloadLen=5, then chunk index=1 payloadLen=0 (final)
- Server receives both chunks with correct payloadLen
- Client2 receives: chunk payloadLen=0, then chunk payloadLen=0

The chunks are being delivered but the payload is lost. The issue is likely in how chunks are being copied or enqueued.

When server does `peer.enqueueResponse(f)`, it's enqueueing the SAME frame pointer. But then it calls `putFrame(f)` which clears the payload!

The frame is being recycled before the peer can read it.


## Sequence Field Not Transmitted

The frame.sequence field is set by setSequence() but NEVER written to the wire in writeFrame() or read in readFrame().

This means isFinal() always returns false on received frames because sequence is always 0.

The spec says sequence is "not in wire format" but then how does the receiver know if a chunk is final?

Solution: Need to transmit sequence in the frame header. The header is 64 bytes:
- 0-1: Version, Type
- 2-3: Flags  
- 4-19: RequestID
- 20-35: ClientID
- 36-51: ChannelID
- 52-59: ClientTimeoutMs
- 60-63: PayloadLength

No space for sequence! Need to add 4 bytes for sequence field, making header 68 bytes.


## Frame Routing Rules

For flagRequest frames:
- startFrame: Acknowledge back to sender ONLY (not routed to peers)
- chunkFrame: Route to peers (with pending receive check)
- endFrame: Acknowledge back to sender ONLY (not routed to peers)

For flagReceive frames:
- All frame types: Check queues and deliver available frames

For flagResponse frames:
- All frame types: Route to target client via responseQueue

The key insight: startFrame and endFrame are control frames that establish/close the request lifecycle. They are acknowledged but not routed. Only chunkFrames (the actual data) are routed between peers.

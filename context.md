# Test Iteration Context - FINAL

## Summary
TestClientBidirectional now passes 10 consecutive runs successfully (0.178s total).

## Issues Fixed

### Iteration 1
**Issue:** Timeout after 10s - deadlock between client1.Send() and client2.Respond()
**Fix:** Added 5s timeout to respondChunkFrame() and respondEndFrame()

### Iteration 2
**Issue:** Still timing out - same deadlock pattern
**Fix:** Changed respondStartFrame timeout from 10s to 5s, inlined time.After() calls to avoid timer leaks

### Iteration 3
**Issue:** CHUNK frames not being received by client2
**Root Cause:** Server's handle() loop blocks on requestHandler.handle() which prevents it from reading more frames from ctx.reads channel. The reader tries to enqueue CHUNK frame but channel buffer (100) fills up, causing deadlock.
**Fix:** Increased server-side reads channel buffer from 100 to 1000 to prevent blocking

### Iteration 4
**Issue:** Worker receives CHUNK request but context is already done
**Root Cause:** CHUNK frames reuse the request from START frame, but the context timeout from START frame has expired by the time CHUNK is processed
**Fix:** Refresh context (cancel old, create new) for CHUNK frames before sending to worker

### Iteration 5
**Issue:** client2's respondChunkFrame returns errPeerError when receiving END frame
**Root Cause:** respondChunkFrame expects only CHUNK frames, but END frame signals end of stream
**Fix:** 
- Modified respondChunkFrame to return io.EOF when END frame is received
- Modified Respond to send END response directly when respondChunkFrame returns io.EOF (instead of calling respondEndFrame which would timeout)

### Iteration 6
**Issue:** Test hangs on client.Close() waiting for WaitGroup
**Root Cause:** Client.Close() closes channels then waits for WaitGroup, but readers are blocked on IO reads. Connection is only closed after WaitGroup completes, creating deadlock.
**Fix:** Close connection BEFORE waiting for WaitGroup so readers can exit from their IO blocking

## Key Changes Made
1. **socket/client.go**: Added timeouts to respond methods, handle END frame in respondChunkFrame, fixed Close() order
2. **socket/server.go**: Increased reads channel buffer to 1000, refresh context for CHUNK frames
3. **socket/connctx.go**: Removed debug logging
4. **socket/reqworker.go**: Removed debug logging

## Test Result
✅ All 10 consecutive runs passed in 0.178s


# Socket Package Redesign - Multiple Peers Support

## Changes Made

### 1. peers.go - Channel-based indexing
- Added `channels map[uuid.UUID]map[uuid.UUID]*peer` to track all peers per channel
- Added `getChannelPeers(channelID, excludeClientID)` to return all peers on a channel
- Modified `set()` to maintain both client and channel indexes
- Modified `delete()` to clean up both indexes

### 2. connctx.go - Broadcast flag
- Added `flagBroadcast uint16 = 0x02` constant

### 3. client.go - Broadcast support
- Added `Broadcast(r io.Reader, timeout) error` method
- Refactored `Send()` to use internal `send(r, timeout, flags)` method
- Broadcast mode skips response collection

### 4. reqworker.go - Broadcast routing
- Modified `handle()` to check `flagBroadcast` flag
- Routes to all channel peers when broadcast flag is set
- Routes to single peer for normal messages

### 5. server.go - Multi-peer handshake
- Modified handshake to use `getChannelPeers()` instead of `pair()`
- Notifies all existing peers when new peer joins channel
- Each peer receives handshake notification with new peer's ID

## Architecture
- Multiple clients can join same channel
- Broadcast messages to all peers on channel
- Backward compatible with 1:1 pairing (uses first peer)
- Minimal code changes to existing structure

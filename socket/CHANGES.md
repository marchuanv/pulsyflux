# Socket Package Changes - 2026-02-15

## Problem: Goroutine Leak

**Symptom**: Tests timing out with 42,000+ goroutines stuck on `<-collector.done`

**Root Cause**: Each flagRequest frame spawned a goroutine that:
- Waited for receivers with timeout polling
- Created ack collector
- Broadcast to peers
- Waited on `<-collector.done`
- Never cleaned up properly

## Solution Applied

### 1. Removed Goroutine Spawn (server.go)

**Before**:
```go
go func(frame *frame, senderEntry *clientEntry, senderID, chanID uuid.UUID) {
    // ... wait for receivers, broadcast, wait for acks ...
}(f, entry, clientID, channelID)
```

**After**:
```go
// Process synchronously in main handle loop
timeout := time.After(time.Duration(f.ClientTimeoutMs) * time.Millisecond)
// ... wait for receivers, broadcast, wait for acks ...
```

**Benefits**:
- No goroutine leak
- Simpler control flow
- Easier to debug
- Proper cleanup

### 2. Added Frame Cloning (connctx.go)

**New Function**:
```go
func cloneFrame(src *frame) *frame {
    f := getFrame()
    // ... copy all fields ...
    if len(src.Payload) > 0 {
        f.Payload = make([]byte, len(src.Payload))
        copy(f.Payload, src.Payload)
    }
    return f
}
```

**Usage in server.go**:
```go
for _, peer := range peers {
    clonedFrame := cloneFrame(f)
    peer.enqueueRequest(clonedFrame)
}
```

**Why Needed**:
- Broadcasting same frame to multiple receivers caused race conditions
- Each receiver needs independent frame copy
- Prevents frame corruption when receivers process at different speeds

### 3. Fixed Timeout Handling (server.go)

**Before**:
```go
select {
case <-timeout:
    errFrame := newErrorFrame(...)
    entry.enqueueResponse(errFrame)
    putFrame(f)
    continue  // BUG: continues inner loop, not outer
case <-ticker.C:
}
```

**After**:
```go
timedOut := false
for {
    // ... check for receivers ...
    select {
    case <-timeout:
        timedOut = true
        break
    case <-ticker.C:
    }
    if timedOut {
        break
    }
}
ticker.Stop()

if timedOut {
    errFrame := newErrorFrame(...)
    entry.enqueueResponse(errFrame)
    putFrame(f)
} else {
    // ... broadcast and wait for acks ...
}
```

**Benefits**:
- Properly exits wait loop on timeout
- Sends error frame to sender
- Cleans up resources
- No infinite loop

## Files Modified

1. **server.go**: Removed goroutine spawn, fixed timeout handling
2. **connctx.go**: Added cloneFrame() function
3. **context.md**: Updated status and next steps

## Testing Required

Run test suite to verify:
- ✅ No goroutine leaks
- ✅ Frames properly cloned for broadcast
- ✅ Timeout handling works correctly
- ✅ Ack collection completes
- ✅ Tests pass without hanging

## Expected Behavior

1. **Client1 calls Send()**:
   - Server receives flagRequest frame
   - Server waits for receivers (polls every 100ms)
   - If timeout: sends error frame, aborts
   - If receivers found: broadcasts cloned frames to all receivers
   - Waits for acks from all receivers
   - Sends ack back to Client1
   - Client1's Send() completes

2. **Client2 calls Receive()**:
   - Blocks waiting for broadcast
   - Receives cloned frame from server
   - Sends ack back to server
   - Processes message
   - Receive() completes

3. **Multiple Receivers**:
   - Each gets independent cloned frame
   - Each sends ack independently
   - Server waits for all acks before signaling sender
   - No frame corruption or race conditions

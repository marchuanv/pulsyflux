# Change Request

## Completed: Transactional Wait() Pattern ✅

### What Was Done

**Problem Solved:**
Deadlock from delayed acknowledgments - `processIncoming()` was waiting for consumer to call `Receive()`/`Respond()` before sending acks.

**Solution:**
- `processIncoming()` sends acknowledgments immediately after receiving each frame
- `Wait()` made transactional - can be called multiple times on same client
- Removed `sync.Once` from `Wait()` to allow repeated calls
- `opErrors` channel stays open for multiple transaction cycles

**Result:**
- Sequential operations work: `Send()` → `Wait()` → `Send()` → `Wait()`
- All tests passing including `TestClientSequentialRequests`
- Clean transactional API for socket consumers

---

## Completed: Unified Stream API with Callbacks ✅

### What Was Done

**Problem Solved:**
Three separate methods (`Send`, `Receive`, `Respond`) with async operations and explicit `Wait()` calls created complexity. Tests required goroutines and channels for coordination.

**Solution:**
Consolidated into a single async `Stream(in io.Reader, out io.Writer, onComplete func(error))` method:
- `Stream(in, nil, callback)` - Send only (async with callback)
- `Stream(nil, out, callback)` - Receive only (async with callback)
- `Stream(in, out, callback)` - Respond pattern (async with callback)

**Implementation Details:**
- Async non-blocking operation - returns immediately
- Callback invoked when operation completes with error (or nil)
- Creates new session on each call to prevent message leakage
- Checks for unconsumed messages from previous session
- Added `sessionMu` for thread-safe session management
- No Wait() needed - callback handles completion

**Result:**
- Simpler API: one method instead of four (`Send`, `Receive`, `Respond`, `Wait`)
- Cleaner tests: no goroutines or channels needed, just callbacks
- Safer: automatic session isolation per call

### Usage Examples

**Before:**
```go
// Send
client.Send(strings.NewReader("hello"))
err := client.Wait()

// Receive
var buf bytes.Buffer
client.Receive(&buf)
err := client.Wait()

// Respond
var reqBuf bytes.Buffer
client.Respond(&reqBuf, strings.NewReader("response"))
err := client.Wait()
```

**After:**
```go
// Send (async with callback)
client.Stream(strings.NewReader("hello"), nil, func(err error) {
    if err != nil {
        log.Printf("Send failed: %v", err)
    }
})

// Receive (async with callback)
var buf bytes.Buffer
client.Stream(nil, &buf, func(err error) {
    if err != nil {
        log.Printf("Receive failed: %v", err)
    }
    // Process buf.Bytes()
})

// Respond (async with callback)
var reqBuf bytes.Buffer
client.Stream(strings.NewReader("response"), &reqBuf, func(err error) {
    if err != nil {
        log.Printf("Respond failed: %v", err)
    }
})
```

### Benefits

1. **Simpler**: One method with clear semantics
2. **Async**: Non-blocking, returns immediately
3. **Test-friendly**: No goroutines or channels in tests, just callbacks
4. **Intuitive**: Parameters determine behavior (in=send, out=receive)
5. **Consistent**: All operations follow same callback pattern
6. **No Wait()**: Callback handles completion notification

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

## Completed: Unified Stream API ✅

### What Was Done

**Problem Solved:**
Three separate methods (`Send`, `Receive`, `Respond`) with async operations and explicit `Wait()` calls created complexity and required careful coordination.

**Solution:**
Consolidated into a single synchronous `Stream(in io.Reader, out io.Writer) error` method:
- `Stream(in, nil)` - Send only (replaces `Send` + `Wait`)
- `Stream(nil, out)` - Receive only (replaces `Receive` + `Wait`)
- `Stream(in, out)` - Respond pattern (replaces `Respond` + `Wait`)

**Implementation Details:**
- Synchronous blocking operation - returns error directly
- Creates new session on each call to prevent message leakage
- Checks for unconsumed messages from previous session
- Removed `opWg` and `opErrors` - no longer needed for async tracking
- Added `sessionMu` for thread-safe session management

**Result:**
- Simpler API: one method instead of four (`Send`, `Receive`, `Respond`, `Wait`)
- Cleaner code: no manual async coordination
- Safer: automatic session isolation per call
- All tests updated and passing

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
// Send
err := client.Stream(strings.NewReader("hello"), nil)

// Receive
var buf bytes.Buffer
err := client.Stream(nil, &buf)

// Respond
var reqBuf bytes.Buffer
err := client.Stream(strings.NewReader("response"), &reqBuf)
```

### Benefits

1. **Simpler**: One method with clear semantics
2. **Safer**: Automatic session management prevents bugs
3. **Cleaner**: No async coordination needed
4. **Intuitive**: Parameters determine behavior (in=send, out=receive)
5. **Consistent**: All operations follow same pattern

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

## New Request: True Stream-Like Experience for Socket Consumers

### Goal
Provide a stream-oriented API that feels natural for continuous message processing, similar to reading from a file or network stream.

### Current State
- Operations are transactional with explicit `Wait()` calls
- Each `Receive()` requires a separate call and `Wait()`
- No built-in loop for continuous message consumption

### Proposed Enhancement

Add streaming methods that handle multiple messages automatically:

```go
// Stream continuously receives messages until error or close
func (c *Client) Stream(handler func([]byte) error) error

// StreamRespond continuously receives and responds
func (c *Client) StreamRespond(handler func([]byte) ([]byte, error)) error
```

### Use Cases

**Current approach** (transactional):
```go
for i := 0; i < 10; i++ {
    var buf bytes.Buffer
    client.Receive(&buf)
    if err := client.Wait(); err != nil {
        return err
    }
    process(buf.Bytes())
}
```

**Proposed approach** (streaming):
```go
err := client.Stream(func(data []byte) error {
    return process(data)
})
```

### Benefits

1. **Simpler code**: No manual loop management
2. **Automatic error handling**: Stream stops on first error
3. **Resource efficient**: Single goroutine handles all messages
4. **Familiar pattern**: Similar to HTTP handlers, gRPC streams
5. **Backward compatible**: Existing transactional API unchanged

### Implementation Notes

- Stream methods would be blocking (like `Wait()`)
- Handler called for each incoming message
- Stream ends on handler error, client close, or context cancellation
- Could add context support: `StreamContext(ctx, handler)`

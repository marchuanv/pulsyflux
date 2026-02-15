# Socket Package API Change Request

## Status: IMPLEMENTED (Reconnection: IN PROGRESS)

## Overview

The socket package client API has been refactored from a single unified `Stream()` method to two dedicated methods: `BroadcastStream()` and `SubscriptionStream()`. This change improves API clarity, usability, and aligns with pub-sub terminology.

## API Changes

### Old API (Deprecated)

```go
// Unified method with parameter-based behavior
func (c *Client) Stream(in io.Reader, out io.Writer, onComplete func(error))

// Usage patterns:
client.Stream(reader, nil, callback)      // Send mode
client.Stream(nil, writer, callback)      // Receive mode  
client.Stream(reader, writer, callback)   // Respond mode
```

**Problems:**
- Ambiguous parameter combinations
- Not self-documenting
- Requires remembering which nil means what
- Doesn't match pub-sub terminology

### New API (Current)

```go
// Dedicated broadcast method
func (c *Client) BroadcastStream(in io.Reader, onComplete func(error))

// Dedicated subscription method
func (c *Client) SubscriptionStream(out io.Writer, onComplete func(error))
```

**Benefits:**
- Clear intent from method name
- Simpler signatures (only relevant parameters)
- Self-documenting code
- Matches pub-sub terminology (broadcast/subscribe)
- Easier to discover and use

## Usage Examples

### Broadcasting Messages

```go
client, _ := NewClient("127.0.0.1:9090", channelID)

var wg sync.WaitGroup
wg.Add(1)
client.BroadcastStream(strings.NewReader("hello"), func(err error) {
    if err != nil {
        log.Printf("Broadcast failed: %v", err)
    }
    wg.Done()
})
wg.Wait()
```

### Subscribing to Messages

```go
client, _ := NewClient("127.0.0.1:9090", channelID)

var buf bytes.Buffer
var wg sync.WaitGroup
wg.Add(1)
client.SubscriptionStream(&buf, func(err error) {
    if err != nil {
        log.Printf("Receive failed: %v", err)
    }
    fmt.Println(buf.String())
    wg.Done()
})
wg.Wait()
```

### Request-Response Pattern

```go
clientA, _ := NewClient("127.0.0.1:9090", channelID)
clientB, _ := NewClient("127.0.0.1:9090", channelID)

var wg sync.WaitGroup
var responseA bytes.Buffer
var requestB bytes.Buffer

wg.Add(2)

// ClientB: receive request, then send response
clientB.SubscriptionStream(&requestB, func(err error) {
    // Process request
    clientB.BroadcastStream(strings.NewReader("response"), func(err error) {
        wg.Done()
    })
})

// ClientA: send request, then receive response
clientA.BroadcastStream(strings.NewReader("request"), func(err error) {
    clientA.SubscriptionStream(&responseA, func(err error) {
        wg.Done()
    })
})

wg.Wait()
```

## Implementation Details

### Code Consolidation

The client code has been consolidated to reduce duplication:

1. **Extracted `connect()` method** - Handles both initial connection and reconnection
2. **Simplified `NewClient()`** - Creates Client struct and calls `connect()`
3. **Removed unused methods** - Cleaned up `receiveFrame()` which was never used
4. **Grouped related methods** - Send methods together, routing methods together

### Idle Management

**Idle Monitor:**
- Checks activity every 1 second
- Closes connection after 5 seconds of inactivity
- Exits when `ctx.closed` is signaled (not `c.done`)
- Each `BroadcastStream()` or `SubscriptionStream()` call resets idle timer

**Activity Tracking:**
- `updateActivity()` called at start of each operation
- Updates `lastActive` timestamp
- Thread-safe with `activityMu` mutex

### Reconnection Feature (IN PROGRESS)

**Goal:** Automatically reconnect when operations are called on closed clients.

**Implementation:**
```go
func (c *Client) ensureConnected() error {
    c.connMu.Lock()
    defer c.connMu.Unlock()
    
    select {
    case <-c.done:
        // Wait for old goroutines to finish
        if c.ctx != nil {
            c.ctx.wg.Wait()
            c.wg.Wait()
        }
        return c.reconnect()
    default:
        return nil
    }
}

func (c *Client) reconnect() error {
    // 1. Create new TCP connection
    // 2. Generate new client ID
    // 3. Recreate all channels
    // 4. Reset WaitGroups
    // 5. Start new goroutines
    // 6. Send registration frame
    // 7. Wait for registration ack
}
```

**Current Status:**
- ✅ Infrastructure in place
- ✅ Basic reconnection logic implemented
- ✅ Goroutine cleanup before reconnect
- ✅ New client ID generation
- ✅ Re-registration with server
- ⚠️ Timing/synchronization issues in edge cases
- ⚠️ Test `TestClientReconnectAfterIdle` currently skipped

**Known Issues:**
1. Race condition between idle monitor closing and reconnection starting
2. WaitGroup synchronization across reconnection boundary
3. Channel lifecycle management during transition

**Workaround:**
Users can create new Client instances instead of reusing closed ones:
```go
// Instead of relying on reconnection
client, _ := NewClient(addr, channelID)
// ... use client ...
// After idle timeout, create new client
client, _ = NewClient(addr, channelID)
```

## Performance Impact

### Benchmark Results (Request-Response Pattern)

| Payload Size | Latency | Throughput | Memory/op | Allocs/op |
|--------------|---------|------------|-----------|-----------|
| 4 bytes      | 592 µs  | 1,689/sec  | 75 KB     | 193       |
| 1 KB         | 605 µs  | 1,652/sec  | 97 KB     | 193       |
| 64 KB        | 997 µs  | 1,003/sec  | 1.6 MB    | 203       |
| 1 MB         | 5.8 ms  | 173/sec    | 18 MB     | 217       |

**Key Observations:**
- Sub-millisecond latency for payloads < 64KB
- Consistent allocation count (~193-217) shows effective buffer pooling
- Linear scaling with payload size
- No performance regression from API change

### Memory Tests - All Passed

| Test | Operations | Result |
|------|-----------|--------|
| TestNoMemoryLeak | 100 req-resp | < 10MB growth |
| TestMemoryLeakUnderLoad | 500 req-resp | < 10MB growth |
| TestMemoryStressLargePayloads | 10 × 5MB | No leaks |
| TestMemoryStressConcurrent | 50 req-resp | No leaks |
| TestNoGoroutineLeak | 10 cycles | ≤ 2 goroutines |

## Migration Guide

### For Existing Code

**Before:**
```go
// Send
client.Stream(strings.NewReader("data"), nil, callback)

// Receive
client.Stream(nil, &buf, callback)

// Respond
client.Stream(strings.NewReader("response"), &reqBuf, callback)
```

**After:**
```go
// Broadcast
client.BroadcastStream(strings.NewReader("data"), callback)

// Subscribe
client.SubscriptionStream(&buf, callback)

// Request-Response (two separate calls)
client.SubscriptionStream(&reqBuf, func(err error) {
    // Process request
    client.BroadcastStream(strings.NewReader("response"), callback)
})
```

### Breaking Changes

1. **Method renamed:** `Stream()` → `BroadcastStream()` / `SubscriptionStream()`
2. **Parameter change:** Separate methods instead of parameter combinations
3. **Respond pattern:** Now requires two method calls instead of one

### Non-Breaking Changes

- Callback signature unchanged: `func(error)`
- Async behavior unchanged
- Session management unchanged
- Error handling unchanged
- Frame protocol unchanged

## Test Coverage

### Updated Tests (18 total)

**Basic Functionality:**
- ✅ TestClientConnectionRefused
- ✅ TestClientEmptyPayload
- ✅ TestClientLargePayload (5MB)
- ✅ TestClientBoundaryPayloadSizes (9 sizes)
- ✅ TestClientSequentialRequests
- ✅ TestClientMultipleConcurrent
- ✅ TestClientRequestResponse
- ✅ TestClientReadError
- ✅ TestTimeoutNoReceivers

**Memory & Performance:**
- ✅ TestNoGoroutineLeak
- ✅ TestNoMemoryLeak
- ✅ TestConnectionCleanup
- ✅ TestMemoryStressLargePayloads
- ✅ TestMemoryStressConcurrent
- ✅ TestMemoryPoolEfficiency
- ✅ TestMemoryLeakUnderLoad

**Operation Order:**
- ✅ TestOperationOrderIndependence (6 subtests)

**Benchmarks:**
- ✅ BenchmarkSendReceive (4 bytes)
- ✅ BenchmarkSendReceive_1KB
- ✅ BenchmarkSendReceive_64KB
- ✅ BenchmarkSendReceive_1MB

**Skipped:**
- ⏭️ TestClientSendAfterClose (no public close method)
- ⏭️ TestClientMultipleClose (no public close method)
- ⏭️ TestTimeoutWithSlowReceivers (immediate ack design)
- ⏭️ TestClientReconnectAfterIdle (reconnection needs work)

## Remaining Work

### High Priority

1. **Fix reconnection edge cases:**
   - Resolve timing issues between idle monitor and reconnection
   - Ensure proper WaitGroup synchronization across reconnection
   - Handle channel lifecycle during transition
   - Add comprehensive reconnection tests

### Medium Priority

2. **Documentation updates:**
   - Update README.md with new API examples
   - Add migration guide for existing users
   - Document reconnection behavior when implemented

### Low Priority

3. **Potential optimizations:**
   - Consider connection pooling for high-frequency reconnections
   - Evaluate if idle timeout should be configurable
   - Consider adding connection state callbacks

## Conclusion

The API change from `Stream()` to `BroadcastStream()`/`SubscriptionStream()` is a significant improvement in clarity and usability. All core functionality works correctly with excellent performance and no memory leaks. The reconnection feature is partially implemented and needs additional work to handle edge cases properly.

**Recommendation:** Merge the API changes now. Complete reconnection feature in a follow-up PR with dedicated focus on goroutine lifecycle management.

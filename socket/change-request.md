# Change Request: Wait() Pattern for Async Operations

## Status: PRODUCTION-READY

### Core Innovation: Wait() Method

All operations are async and return immediately. `Wait()` provides the synchronization point:

```go
// Operations return immediately (void)
client.Send(data)      // No return value
client.Receive(&buf)   // No return value
client.Respond(&req, resp)  // No return value

// Wait() blocks until ALL operations complete
err := client.Wait()   // Returns first error or nil
```

## Why Wait() Matters

1. **Non-blocking API**: Fire operations without waiting
2. **Concurrent Operations**: Multiple operations in flight simultaneously
3. **Centralized Error Handling**: All errors collected, first returned
4. **Thread-Safe**: Can only be called once per client (sync.Once)
5. **Simple Testing**: Easy to test timeouts, errors, concurrent scenarios

## Wait() Implementation

```go
type Client struct {
    opWg      sync.WaitGroup  // Tracks all async operations
    opErrors  chan error      // Collects errors (buffered 100)
    waitOnce  sync.Once       // Ensures Wait() called only once
}

func (c *Client) Wait() error {
    var firstErr error
    c.waitOnce.Do(func() {
        c.opWg.Wait()           // Block until all ops complete
        close(c.opErrors)       // Close error channel
        for err := range c.opErrors {  // Drain all errors
            if err != nil && firstErr == nil {
                firstErr = err  // Capture first error
            }
        }
    })
    return firstErr
}
```

### Key Features

- **Thread-Safe**: `sync.Once` ensures single execution
- **Blocks Until Complete**: `opWg.Wait()` waits for all operations
- **Error Collection**: Drains all errors from channel
- **First Error**: Returns first non-nil error
- **Channel Cleanup**: Closes `opErrors` after draining

## Usage Examples

### Basic Pattern
```go
client.Send(data)
err := client.Wait()  // Block until send completes
if err != nil {
    log.Fatal(err)
}
```

### Multiple Operations
```go
client.Send(data1)
client.Send(data2)
client.Receive(&buf)
err := client.Wait()  // Wait for all 3 operations
```

### Multiple Clients
```go
client1.Send(data)
client2.Receive(&buf)

if err := client1.Wait(); err != nil {
    log.Fatal(err)
}
if err := client2.Wait(); err != nil {
    log.Fatal(err)
}
```

## Testing Benefits

```go
// Test timeout
publisher.Send(data)
err := publisher.Wait()
if !strings.Contains(err.Error(), "timeout") {
    t.Error("Expected timeout")
}

// Test concurrent operations
client1.Send(data)
client2.Receive(&buf)
var wg sync.WaitGroup
wg.Add(2)
go func() { defer wg.Done(); client1.Wait() }()
go func() { defer wg.Done(); client2.Wait() }()
wg.Wait()
```

## Implementation Details

### Async Operations

All operations spawn goroutines:

```go
func (c *Client) Send(r io.Reader) error {
    c.opWg.Add(1)
    go func() {
        defer c.opWg.Done()
        if err := c.doSend(r); err != nil {
            select {
            case c.opErrors <- err:
            default:
            }
        }
    }()
    return nil
}
```

Same pattern for `Receive()` and `Respond()`.

### Error Collection

- Errors sent to `opErrors` channel (buffered 100)
- `Wait()` drains channel and returns first error
- Non-blocking send prevents goroutine leaks if channel full

## Architecture Changes

### Frame Routing
- `routeFrames()` separates request/response frames
- `processIncoming()` assembles requests in background
- No frame stealing between operations

### Delayed Acknowledgment
- Acks sent only after `Receive()`/`Respond()` called
- Enables timeout testing
- Consumer controls ack timing

## API

```go
// All async - no return value
Send(r io.Reader)
Receive(r io.Reader)  // r must be io.Writer
Respond(req io.Reader, resp io.Reader)  // req must be io.Writer

// Synchronization point
Wait() error  // Blocks until all operations complete
```

## Benefits

1. **Flexibility**: Call operations in any order
2. **Concurrency**: Multiple operations without blocking
3. **Error Handling**: Single point to check errors
4. **Testability**: Easy to test all scenarios
5. **Performance**: Better CPU utilization

## Testing

All tests updated to use Wait() pattern:

```go
// Before
client1.Send(data)  // Blocked until complete

// After  
client1.Send(data)  // Returns immediately
err := client1.Wait()  // Block here
```

Tests pass:
- ✅ `TestOperationOrderIndependence`
- ✅ `TestTimeoutNoReceivers`
- ✅ `TestTimeoutWithSlowReceivers`
- ✅ `TestTimeoutNoResponse`
- ✅ All existing tests

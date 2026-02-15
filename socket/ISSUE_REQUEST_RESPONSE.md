# Request-Response Pattern Issue

## Problem

The current implementation does not support a proper 2-client request-response pattern where:
1. ClientA sends a request
2. ClientB receives the request and sends a response
3. ClientA receives the response

## Root Cause

The `Stream()` method recreates the session on each call, but the background `processIncoming()` goroutine continues running and writes incoming messages to the old session's `incoming` channel. When ClientA tries to call `Stream()` a second time to receive the response, it detects "session has unconsumed messages" and fails.

## Current Workarounds

### Option 1: Use 3 clients
```go
// ClientA sends request
clientA.Stream(strings.NewReader("ping"), nil, callback)

// ClientB receives and responds
clientB.Stream(strings.NewReader("pong"), &requestBuf, callback)

// ClientC receives response
clientC.Stream(nil, &responseBuf, callback)
```

### Option 2: Both clients use respond mode (DEADLOCK)
```go
// Both wait to receive first - neither sends initially
clientA.Stream(strings.NewReader("ping"), &responseBuf, callback) // Waits for incoming
clientB.Stream(strings.NewReader("pong"), &requestBuf, callback)  // Waits for incoming
// DEADLOCK: Both waiting, neither sending
```

## Proposed Solutions

### Solution 1: Allow multiple concurrent Stream() calls
- Don't recreate session on each call
- Allow multiple operations to be in flight
- Each operation gets its own request/response tracking

### Solution 2: Add dedicated Request() method
```go
func (c *Client) Request(in io.Reader, out io.Writer, onComplete func(error)) {
    // Send request from 'in'
    // Wait for and receive response to 'out'
    // Single atomic operation
}
```

### Solution 3: Fix session management
- Properly drain old session before creating new one
- Or make session reference atomic and update processIncoming() to always write to current session
- Requires careful synchronization

### Solution 4: Return response in callback
```go
client.Stream(strings.NewReader("ping"), nil, func(err error, response []byte) {
    // Response automatically captured if one arrives
})
```

## Test Case

See `TestClientRequestResponse` in `socket_test.go` - currently fails with "session has unconsumed messages"

## Impact

This limits the socket package to:
- One-way broadcast (send-only)
- One-way receive (receive-only)
- Single respond operation (receive-then-send in one call)

It does NOT support:
- Send-then-receive as separate operations
- Multiple sequential request-response cycles from same client
- True 2-client ping-pong pattern

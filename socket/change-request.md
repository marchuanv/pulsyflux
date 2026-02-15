# Change Request: Remove Operation Ordering Requirement

## Problem Statement

The original implementation required tests to call `Send()`, `Receive()`, or `Respond()` in a specific order due to serialization via `opMu` lock. This made the API inflexible and tests fragile, as operations had to be carefully orchestrated.

## Root Cause

The previous design used:
1. **Operation Mutex (`opMu`)**: Serialized all `Send()`, `Receive()`, and `Respond()` calls on the same client
2. **Shared Channel (`ctx.reads`)**: Both outgoing operations (waiting for acks) and incoming operations (receiving requests) competed for frames from the same channel
3. **Blocking Behavior**: `Receive()` had to be called before frames arrived, or frames would be consumed by other operations

## Solution

Refactored the client infrastructure to support concurrent, order-independent operations:

### 1. Removed Operation Serialization
- **Removed**: `opMu` mutex that serialized operations
- **Result**: `Send()`, `Receive()`, and `Respond()` can now be called concurrently without blocking each other

### 2. Added Background Request Processing
- **Added**: `processIncomingRequests()` goroutine that runs continuously in the background
- **Purpose**: Automatically receives and assembles incoming requests from other clients
- **Behavior**: 
  - Listens for incoming request frames (with `flagRequest`)
  - Sends acknowledgments for each frame
  - Assembles complete requests (start → chunks → end)
  - Queues assembled requests in `incoming` channel

### 3. Separated Request and Response Channels
- **Added**: `ctx.requests` channel for incoming request frames (with `flagRequest`)
- **Added**: `ctx.responses` channel for response frames (acks, errors without `flagRequest`)
- **Added**: `routeFrames()` goroutine to route frames based on flags
- **Result**: Request frames and response frames no longer compete for the same channel

### 4. Simplified Public API Methods

**`Receive(incoming chan io.Reader)`**:
- Now simply reads from pre-assembled `incoming` channel
- Non-blocking - returns immediately if request is available
- No longer needs to coordinate with `Send()` timing

**`Respond(incoming, outgoing chan io.Reader)`**:
- Reads pre-assembled request from `incoming` channel
- Waits for response from `outgoing` channel
- Calls `Send()` to broadcast response (no lock conflict)

**`Send(r io.Reader)`**:
- Unchanged externally
- Internally uses `ctx.responses` channel for acks (not `ctx.reads`)

## Architecture Changes

### Before (Serialized)
```
Client
├── opMu (serializes all operations)
├── ctx.reads (shared by Send and Receive)
├── Send() - locks opMu, sends frames, waits for acks
├── Receive() - locks opMu, receives frames, sends acks
└── Respond() - locks opMu, calls Receive() then Send()
```

### After (Concurrent)
```
Client
├── ctx.requests (incoming request frames)
├── ctx.responses (acks and errors)
├── processIncomingRequests() - background goroutine
│   ├── Reads from ctx.requests
│   ├── Sends acks
│   └── Queues to incoming channel
├── Send() - sends frames, waits for acks from ctx.responses
├── Receive() - reads from incoming channel (non-blocking)
└── Respond() - reads from incoming, sends response

connctx
└── routeFrames() - routes frames by flag
    ├── flagRequest → ctx.requests
    └── flagAck/flagNone → ctx.responses
```

## Code Changes

### client.go

**Added**:
- `incoming chan *assembledRequest` - buffered channel (16) for assembled requests
- `assembledRequest` struct with `payload []byte`
- `processIncomingRequests()` - background goroutine
- `receiveRequestFrame()` - reads from `ctx.requests` channel
- `wg sync.WaitGroup` - tracks background goroutine

**Removed**:
- `opMu sync.Mutex` - no longer needed
- `receiveAssembledChunkFramesWithAck()` - logic moved to background goroutine

**Modified**:
- `NewClient()` - starts `processIncomingRequests()` goroutine, initializes new channels
- `Send()` - removed `opMu.Lock()`, uses `ctx.responses` via `waitForAck()`
- `Receive()` - simplified to read from `incoming` channel
- `Respond()` - simplified to read from `incoming` channel then call `Send()`
- `waitForAck()` - reads from `ctx.responses` instead of `ctx.reads`
- `Close()` - waits for background goroutine, closes `incoming` channel

### connctx.go

**Added**:
- `requests chan *frame` - channel for incoming request frames
- `responses chan *frame` - channel for response frames (acks, errors)
- `routeFrames()` - goroutine that routes frames based on `flagRequest`

**Modified**:
- `connctx` struct - added `requests` and `responses` channels

## Benefits

1. **Test Flexibility**: Tests no longer need to call operations in specific order
2. **Concurrent Operations**: Multiple goroutines can call `Send()`, `Receive()`, `Respond()` simultaneously
3. **Automatic Request Handling**: Incoming requests are automatically received and queued
4. **Cleaner Separation**: Request processing and response handling are completely independent
5. **No Frame Stealing**: Frames are routed to the correct channel, preventing competition

## Testing Impact

### Before
```go
// HAD to call Receive() before Send() to avoid frame stealing
go func() {
    client2.Receive(incoming)  // Must be called first
}()
time.Sleep(50 * time.Millisecond)  // Ensure Receive is waiting
client1.Send(data)  // Now safe to send
```

### After
```go
// Can call in any order - background goroutine handles incoming frames
client1.Send(data)  // Send first
time.Sleep(10 * time.Millisecond)  // Optional - just for test timing
go func() {
    client2.Receive(incoming)  // Can be called anytime
}()
```

## Backward Compatibility

The public API remains unchanged:
- `Send(r io.Reader) error`
- `Receive(incoming chan io.Reader) error`
- `Respond(incoming, outgoing chan io.Reader) error`

Existing code using these methods will continue to work, but will now benefit from the ability to call them in any order.

## Performance Considerations

**Added Overhead**:
- One additional goroutine per client (`processIncomingRequests`)
- One additional goroutine per client (`routeFrames`)
- Two additional buffered channels per client (`requests`, `responses`)
- One additional buffered channel per client (`incoming`)

**Benefits**:
- Eliminates lock contention from `opMu`
- Reduces blocking between operations
- Better CPU utilization with concurrent operations

## Future Enhancements

Potential improvements:
1. Add timeout support to `Receive()` and `Respond()`
2. Add context.Context support for cancellation
3. Add metrics for queue depths and processing times
4. Consider making `incoming` channel size configurable

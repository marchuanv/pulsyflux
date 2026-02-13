# Socket Implementation Context

## Current Status (Updated)

### Test Status
- All main tests still failing after flow simplification attempt
- Need to ensure response delivery is reliable

### Current Implementation

The flow has been simplified to:
1. **Request frames (flagRequest)**: Try pending receives first, then enqueue to all peers/self
2. **Receive frames (flagReceive)**: Dequeue from own/peer queues, or keep pending if nothing available  
3. **Response frames (flagResponse)**: Route to specific client by ClientID

### Key Issue

Response frames are being enqueued to responseQueue, but the processResponses goroutine uses `connctx.enqueue` which can drop frames if the channel is full. This causes responses to be lost.

### Solution Needed

Change processResponses to use blocking sends instead of enqueue:
```go
func (e *clientEntry) processResponses() {
    for f := range e.responseQueue {
        select {
        case e.connctx.writes <- f:
        case <-e.connctx.closed:
            putFrame(f)
            return
        }
    }
}
```

### Next Steps
1. Fix processResponses to use blocking delivery
2. Test all scenarios
3. Handle edge cases (queue full, client disconnect, etc.)

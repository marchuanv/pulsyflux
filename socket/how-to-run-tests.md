# How to Run Tests

## Test Strategy

Run tests **one at a time** with a **10-second timeout** to detect hangups and infinite loops.

## Current Issue

Tests hang because Client1's Send() waits for acks but server is stuck waiting at `<-collector.done`.

**Logs show**:
1. Client1 sends message (reqID=X)
2. Server broadcasts to Client2
3. Client2 receives and should send ack
4. But Client2 sends REQUEST frame instead of ACK frame
5. Server broadcasts Client2's request back to Client1
6. Ping-pong loop

**Root Cause**: Acks are not being sent or received properly. Server waits forever for acks.

## Commands

### Run Single Test
```bash
go test -v -timeout 10s -run TestClientEmptyPayload
```

## Next Steps

1. Verify acks are being sent by Client2
2. Verify server receives and processes acks
3. Verify ackCollector signals done when all acks received

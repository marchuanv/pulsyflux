# Test Pattern Clarification

## Correct Understanding

The tests are **CORRECT** as originally written. The pattern is:

1. **Client1**: Calls `Send()` - broadcasts message and waits for acks
2. **Client2**: Calls `Respond()` - receives message, processes, broadcasts response
3. **Client1's Send()**: Completes after all acks received (does NOT wait for response data)

## Key Insight

`Send()` is **fire-and-forget from application perspective**:
- Internally waits for acks from all receivers
- Does NOT wait for response data
- Returns after acks received, allowing sender to continue

## Test Flow

```
Client1                 Server                  Client2
  |                       |                       |
  +--Send()-------------->|                       |
  | (broadcast msg)       +--broadcast---------->|
  |                       |                       +--Respond()
  | (wait for acks)       |<--ack-----------------| (receive msg)
  |<--ack-----------------+                       |
  | Send() returns        |                       | (process)
  |                       |<--Send()--------------|
  |                       | (broadcast response)  | (broadcast response)
  |                       +--broadcast---------->|
  |                       |                       | Respond() returns
  |                       |<--ack-----------------|
  |                       +--ack----------------->|
  |                       |                       |
```

## Why Tests Hang

The current implementation has Client2's response going back to Client1 as a new broadcast, which Client1 isn't expecting. This creates the ping-pong effect seen in logs.

## Solution Needed

The socket implementation needs to handle the pub-sub pattern correctly where:
- All clients receive all broadcasts (except sender)
- Send() completes after acks, not after response
- No infinite loops when clients respond to each other

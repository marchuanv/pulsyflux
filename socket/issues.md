# Socket Package Issues

## Issue #1: Frame Interleaving with Multiple Concurrent Receivers

**Status**: IDENTIFIED - NOT FIXED

**Test**: TestClientMultipleConcurrent

**Symptom**: "invalid frame" errors when multiple clients send/receive concurrently

### Investigation Log

Added logging to track frame flow:
- Client.Send() logs START/COMPLETE with reqID
- Client.Receive() logs START with origReqID and which request it receives
- receiveStartFrame() logs every frame received with type and reqID

### Key Findings from Logs

```
[Client 31ad578d] Receive: START origReqID=fe20f6dc
[Client 31ad578d] receiveStartFrame: got frame type=4 reqID=95e29baf (expecting reqID=00000000)
[Client 31ad578d] Receive: Got request reqID=95e29baf from client=cbb0c5fc

[Client 8e6bc9b5] Receive: START origReqID=30917783
[Client 8e6bc9b5] receiveStartFrame: got frame type=4 reqID=feed0a1e (expecting reqID=00000000)
[Client 8e6bc9b5] Receive: Got request reqID=feed0a1e from client=8d447ff6
```

Then errors:
```
Receiver 0 error: invalid frame
Receiver 1 error: invalid frame
Receiver 2 error: invalid frame
```

### Root Cause

**Multiple receivers are receiving frames from the SAME requests.**

When multiple clients call Receive() on the same channel:
1. Client 31ad578d receives request 95e29baf
2. Client 8e6bc9b5 receives request feed0a1e
3. But then BOTH clients start receiving frames from BOTH requests
4. This causes "invalid frame" errors because they receive chunk/`end frames from requests they didn't initiate

**The server is broadcasting frames to ALL receivers on a channel instead of routing each request to exactly ONE receiver.**

### Expected Behavior

Each request should be assigned to exactly ONE receiver:
- Request 95e29baf → Client 31ad578d (and ONLY this client)
- Request feed0a1e → Client 8e6bc9b5 (and ONLY this client)
- Request dfc1c1d8 → Client 1c1178b2 (and ONLY this client)

### Server Routing Problem

The server needs to:
1. When a receiver calls Receive(), assign it to handle the NEXT available request
2. Route ALL frames (start/chunk/end) for that request ONLY to that specific receiver
3. Not broadcast frames to other receivers on the channel

### Next Steps

1. Review server.go routing logic for flagRequest frames
2. Check how frames are enqueued to peer queues
3. Ensure each request is assigned to exactly one receiver
4. Add request-to-receiver mapping to prevent frame interleaving

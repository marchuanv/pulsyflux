# Socket Package Rework

## Status: SPECIFICATION FOR REDESIGN

This document describes a complete redesign of the socket package to implement proper pub-sub (broadcast) semantics with acknowledgment-based synchronization.

## Original Issue: Frame Interleaving with Multiple Concurrent Receivers

**Test**: TestClientMultipleConcurrent

**Symptom**: "invalid frame" errors when multiple clients send/receive concurrently

**Root Cause**: The original implementation attempted request-response semantics but the architecture requires pub-sub (broadcast) semantics. The current code has fundamental design issues that cannot be fixed with minor changes.

## Decision: Complete Redesign

After analysis, the socket layer needs to be redesigned from the ground up to properly support:
1. Broadcast pub-sub pattern (not request-response)
2. Frame-level acknowledgment synchronization
3. Self-filtering by ClientID
4. Clean separation between receive-only and receive-respond patterns

## New Architecture Specification

**Socket Layer Design: Pure Pub-Sub (Broadcast)**

The socket layer is a **broadcast-only pub-sub system**:
- `Send()` broadcasts message to ALL other clients on channel (excludes sender)
- `Receive()` receives broadcasts from other clients
- `Respond()` receives broadcasts, processes, and broadcasts response
- Server broadcasts frames to ALL clients EXCEPT the sender
- Clients don't need to filter out their own messages (server handles exclusion)

**RequestID Purpose**:
1. **Server-side**: Group frames together (startFrame→chunkFrame→endFrame for same message)
2. **Client-side**: Identify which message the frames belong to

**NOT for request-response tracking** - there is no 1-to-1 request-response pattern.

### Frame Protocol

**Frame Types**:

**Data Frames** (application-level, carry payload):
- `startFrame (0x04)`: Beginning of a message
- `chunkFrame (0x05)`: Payload data chunks  
- `endFrame (0x06)`: End of a message

**Control Frames** (protocol-level, no payload):
- `errorFrame (0x03)`: Error notification
- `ackFrame (0x07)`: Acknowledgment that a data frame was received (NEW)

**Frame Flags**:
- `flagRequest (0x01)`: Data frame being broadcast
- `flagAck (0x02)`: Control frame - acknowledgment (NEW)
- `flagReceive (0x04)`: DEPRECATED - will be removed
- `flagResponse (0x08)`: Control frame - used for errors

**Critical Distinction**:
- **Data frames** use `flagRequest` and contain application data
- **Control frames** use `flagAck` or `flagResponse` and contain protocol metadata
- **Ack frames** are sent by receivers to acknowledge receipt of data frames
- **Server aggregates acks**: N receivers → N ack frames → 1 ack frame to sender

**Frame Flow Example**:
```
Sender                  Server                  Receiver
  |
  +--startFrame---------->|  (DATA FRAME)
  | flagRequest           +--broadcast------------>|
  | Payload: none          |                         |
  |                       |                         |
  | (blocked)             |                         +--ackFrame---------->
  |                       |<------------------------| (CONTROL FRAME)
  |                       | (aggregate)             | flagAck
  |                       |                         | Payload: none
  |<--ackFrame------------+                         |
  | (CONTROL FRAME)       |                         |
  | flagAck               |                         |
  | Payload: none         |                         |
  | (unblocked)           |                         |
```

**When Each Frame Type is Used**:

**Data Frames** (carry application data):
- `startFrame`: Signals beginning of message, no payload (just metadata)
- `chunkFrame`: Carries actual application data in payload (up to 1MB per chunk)
- `endFrame`: Signals end of message, no payload (just metadata)
- All use `flagRequest` flag
- Sent by application via `Send()` or `Respond()`

**Control Frames** (protocol-level, NO application data):
- `ackFrame`: Acknowledges receipt of a data frame, no payload
  - Sent by receivers after receiving each data frame
  - Sent by server to sender after aggregating receiver acks
  - Uses `flagAck` flag
- `errorFrame`: Reports protocol errors, payload contains error message string
  - Sent by server on timeout or other errors
  - Uses `flagResponse` flag

**Key Rule**: 
- **Application data ONLY goes in chunkFrame payload**
- **Control frames NEVER carry application data** (ack has no payload, error has error message only)
- **start/end frames are data frames but have no payload** (just signal boundaries)

**CRITICAL: Frame-by-Frame Pub-Sub with Acknowledgments**

**From caller perspective**: `Send()` is fire-and-forget (no return value expected)

**Internally**: Each frame requires acknowledgment from ALL receivers before proceeding:

1. **Sender publishes startFrame** → Server broadcasts to ALL receivers
2. **Each receiver gets startFrame** → Checks sequence field → Sends ack
3. **Server collects acks** → Waits for acks from ALL receivers → Signals sender to proceed
4. **Sender publishes chunkFrame** → Server broadcasts to ALL receivers
5. **Each receiver gets chunkFrame** → Checks sequence field → Sends ack
6. **Server collects acks** → Waits for acks from ALL receivers → Signals sender to proceed
7. **Repeat for each chunk**
8. **Sender publishes endFrame** → Server broadcasts to ALL receivers
9. **Each receiver gets endFrame** → Sends ack
10. **Server collects acks** → Waits for acks from ALL receivers → Send() completes

**Sequence Field Purpose**:
- Tells receivers how many frames to expect in a sequence
- `isFinal` flag indicates last frame in sequence
- Receivers use this to know when to send acks

**Example with 3 clients**:
```
ClientA (sender)    Server              ClientB (rcv)       ClientC (rcv)
     |                |                      |                   |
     +--startFrame--->|  (DATA, no payload)  |                   |
     | seq=0,final    +--broadcast---------->+------------------>+
     | clientA        |  (excludes A)        |                   |
     |                |                  (DATA FRAME)        (DATA FRAME)
     |                |                  send ack            send ack
     |                |                      |                   |
     |                | <--ackFrame----------+  (CONTROL)        |
     |                | <--ackFrame--------------------------+  (CONTROL)
     |                |                      |                   |
     | (blocked)      | (got 2/2 acks)       |                   |
     |                | (aggregate)          |                   |
     | (unblocked)    |                      |                   |
     |<--ackFrame-----+  (CONTROL)           |                   |
     |                |                      |                   |
     +--chunkFrame--->|  (DATA, has payload) |                   |
     | seq=0,final    +--broadcast---------->+------------------>+
     | clientA        |  (excludes A)        |                   |
     | Payload: data  |                  (DATA FRAME)        (DATA FRAME)
     |                |                  send ack            send ack
     |                |                      |                   |
     |                | <--ackFrame----------+  (CONTROL)        |
     |                | <--ackFrame--------------------------+  (CONTROL)
     |                |                      |                   |
     | (blocked)      | (got 2/2 acks)       |                   |
     |                | (aggregate)          |                   |
     | (unblocked)    |                      |                   |
     |<--ackFrame-----+  (CONTROL)           |                   |
     |                |                      |                   |
     +--endFrame----->|  (DATA, no payload)  |                   |
     | seq=0,final    +--broadcast---------->+------------------>+
     | clientA        |  (excludes A)        |                   |
     |                |                  (DATA FRAME)        (DATA FRAME)
     |                |                  send ack            send ack
     |                |                      |                   |
     |                | <--ackFrame----------+  (CONTROL)        |
     |                | <--ackFrame--------------------------+  (CONTROL)
     |                |                      |                   |
     | (blocked)      | (got 2/2 acks)       |                   |
     |                | (aggregate)          |                   |
     | (unblocked)    |                      |                   |
     |<--ackFrame-----+  (CONTROL)           |                   |
     | Send() returns |                      |                   |
```

**Server Synchronization**:
- Server tracks active receivers per channel (count = N, excludes sender)
- For each frame from sender:
  - Broadcast to all N receivers (excludes sender)
  - Collect acks from receivers
  - When N acks received → signal sender to proceed
- Ack aggregation: N receivers → N acks → 1 "proceed" to sender

### Message Flow Example

```
ClientA (pub)    Server          ClientB (sub)   ClientC (sub)   ClientD (sub)
    |              |                  |               |               |
    +--Send------->|                  |               |               |
    | reqA,        |                  |               |               |
    | clientA      +--broadcast------>+-------------->+-------------->+
    |              |                  |               |               |
    | (A filters   |           (B,C,D receive reqA from clientA)     |
    |  out reqA    |                  |               |               |
    |  clientA)    |               process         process         process
    |              |                  |               |               |
    |              | <--Send----------+               |               |
    |              |  reqB, clientB   |               |               |
    +<-------------+                  |               |               |
    |              +--------------------------------->+-------------->+
    |              |                  |               |               |
 (A,C,D receive reqB from clientB)   | (B filters    |               |
    |              |                  |  out reqB     |               |
 process        process               |  clientB)  process         process
    |              |                  |               |               |
    +--Send------->|                  |               |               |
    | reqA2,       |                  |               |               |
    | clientA      +--broadcast------>+-------------->+-------------->+
    |              |                  |               |               |
```

**Key points**:
- ALL clients receive ALL messages (broadcast) EXCEPT sender doesn't receive own messages
- Server excludes sender from broadcast
- No client-side filtering needed
- Pure pub-sub pattern

## Analysis of Current Implementation Issues

### Observed Symptoms from Logs

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

### Fundamental Design Issues in Current Implementation

**The current implementation has multiple bugs**:

1. **Send() incorrectly waits for response data**: Send() should wait for acks (synchronization), not response data. Currently it waits for response payload which doesn't exist in pub-sub.

2. **No ack mechanism**: Receivers don't send acks after receiving frames. Server doesn't collect acks or signal sender to proceed.

3. **Receive() uses wrong requestID for chunks**: After getting startFrame with reqID=X, it uses origReqID (receiver's own ID) instead of reqID when receiving chunks.

4. **Receive() sends response with flagResponse**: Should broadcast response with flagRequest to all clients, not send to specific client with flagResponse.

5. **Complex flagReceive polling**: The flagReceive mechanism is unnecessary for pure broadcast - just read from channel.

6. **Server doesn't track receiver count**: Server needs to know how many receivers to wait for acks from.

7. **Server doesn't exclude sender**: Server broadcasts to all clients including sender, requiring client-side filtering.

---

## Redesign Specification: Pub-Sub with Ack-Based Synchronization

**Status**: APPROVED FOR IMPLEMENTATION

### Design Overview

Implement broadcast pub-sub with frame-level acknowledgments:

1. **Send()**: Broadcast each frame, wait for acks from ALL receivers before next frame
2. **Receive()**: Get broadcasts from other clients, send acks, process
3. **Respond()**: Get broadcasts, send acks, process, broadcast response
4. **Server**: Broadcast frames to all clients EXCEPT sender, collect acks, signal sender when all acks received
5. **Client**: No self-filtering needed (server excludes sender)

### New API Design

**Before**:
```go
// Request-response pattern (WRONG)
response, err := client.Send(request, timeout)

// Combined receive and respond
err := client.Receive(incoming, outgoing, timeout)
```

**After**:
```go
// Pub-sub with ack synchronization (CORRECT)
err := client.Send(request)  // Fire-and-forget from caller, but internally waits for acks

// Separated receive and respond
err := client.Receive(incoming)  // Just receive and ack
err := client.Respond(incoming, outgoing)  // Receive, process, and respond
```

### Implementation

**1. Server-Side: Ack Collection and Synchronization**

**Add to registry**:
```go
type registry struct {
    mu              sync.RWMutex
    clients         map[uuid.UUID]*clientEntry
    channels        map[uuid.UUID]map[uuid.UUID]*clientEntry
    pendingAcks     map[uuid.UUID]*ackCollector  // NEW: track acks per frame
}

type ackCollector struct {
    frameID       uuid.UUID  // Unique ID for this frame instance
    requestID     uuid.UUID  // The message requestID
    senderID      uuid.UUID  // Who sent the frame
    frameType     uint8      // start/chunk/end
    expectedAcks  int        // Number of receivers
    receivedAcks  int        // Acks received so far
    done          chan struct{}  // Signal when all acks received
    mu            sync.Mutex
}
```

**Server routing logic**:
```go
func (s *Server) handle(conn net.Conn) {
    // ... existing setup ...
    
    for {
        select {
        case f, ok := <-ctx.reads:
            if !ok {
                return
            }
            
            if f.Flags&flagRequest != 0 {
                // Sender publishing a frame
                peers := s.registry.getChannelPeers(entry.channelID, clientID)
                
                if len(peers) == 0 {
                    // No receivers, wait for receivers with timeout
                    timeout := time.After(time.Duration(f.ClientTimeoutMs) * time.Millisecond)
                    ticker := time.NewTicker(100 * time.Millisecond)
                    defer ticker.Stop()
                    
                    for {
                        select {
                        case <-timeout:
                            // Timeout - send error frame
                            errFrame := createErrorFrame(f, "no receivers available")
                            entry.enqueueResponse(errFrame)
                            putFrame(f)
                            continue  // Continue to next frame
                        case <-ticker.C:
                            // Check if receivers joined
                            peers = s.registry.getChannelPeers(entry.channelID, clientID)
                            if len(peers) > 0 {
                                goto broadcast  // Receivers available, proceed to broadcast
                            }
                        }
                    }
                }
                
            broadcast:
                // Create ack collector
                frameID := uuid.New()
                collector := s.registry.createAckCollector(frameID, f.RequestID, clientID, f.Type, len(peers))
                
                // Broadcast to all receivers (excludes sender)
                for _, peer := range peers {
                    peer.enqueueRequest(f)
                }
                
                // Wait for all acks (blocking)
                <-collector.done
                
                // Signal sender to proceed
                entry.enqueueResponse(createAckFrame(f))
                
                // Cleanup
                s.registry.removeAckCollector(frameID)
            } else if f.Flags&flagAck != 0 {
                // Receiver sending ack
                s.registry.recordAck(f.FrameID, clientID)
            } else if f.Flags&flagResponse != 0 {
                // Response routing (existing)
                if peer, ok := s.registry.get(f.ClientID); ok {
                    peer.enqueueResponse(f)
                } else {
                    putFrame(f)
                }
            }
        }
    }
}
```

**2. Client-Side: Send() with Ack Waiting**

```go
func (c *Client) Send(r io.Reader) error {
    if !c.opMu.TryLock() {
        return errOperationInProgress
    }
    defer c.opMu.Unlock()

    reqID := uuid.New()
    timeoutMs := uint64(defaultTimeout.Milliseconds())

    // Send startFrame and wait for ack
    if err := c.sendStartFrame(reqID, c.clientID, timeoutMs, flagRequest); err != nil {
        return err
    }
    if err := c.waitForAck(reqID, startFrame); err != nil {
        return err
    }

    // Send chunks and wait for ack after each
    chunks, err := c.disassembleChunks(r)
    if err != nil {
        return err
    }
    for i, chunk := range chunks {
        isFinal := i == len(chunks)-1
        if err := c.sendChunkFrame(reqID, c.clientID, timeoutMs, i, isFinal, chunk, flagRequest); err != nil {
            return err
        }
        if err := c.waitForAck(reqID, chunkFrame); err != nil {
            return err
        }
    }

    // Send endFrame and wait for ack
    if err := c.sendEndFrame(reqID, c.clientID, timeoutMs, flagRequest); err != nil {
        return err
    }
    if err := c.waitForAck(reqID, endFrame); err != nil {
        return err
    }

    return nil  // All frames sent and acked
}

func (c *Client) waitForAck(reqID uuid.UUID, frameType uint8) error {
    // Wait for ack frame (CONTROL FRAME) from server
    for {
        select {
        case f := <-c.ctx.reads:
            if f.Type == ackFrame && f.RequestID == reqID {
                putFrame(f)
                return nil
            }
            if f.Type == errorFrame && f.RequestID == reqID {
                // Server sent error (e.g., "no receivers available" on timeout)
                // Extract error message from frame payload
                errMsg := string(f.Payload)
                putFrame(f)
                return fmt.Errorf("server error: %s", errMsg)
            }
            // Put back non-matching frames
            putFrame(f)
        case <-c.ctx.closed:
            return errClosed
        }
    }
}
```

**3. Client-Side: Receive() and Respond() Methods**

```go
// Receive() - Just receive and ack, no response
func (c *Client) Receive(incoming chan io.Reader) error {
    if !c.opMu.TryLock() {
        return errOperationInProgress
    }
    defer c.opMu.Unlock()

    // Wait for broadcast startFrame (server excludes our own messages)
    rcvF, err := c.receiveStartFrame(uuid.Nil)
    if err != nil {
        return err
    }

    reqID := rcvF.RequestID
    if err := c.sendAck(reqID, startFrame); err != nil {
        putFrame(rcvF)
        return err
    }
    putFrame(rcvF)

    reqBuf, err := c.receiveAssembledChunkFramesWithAck(reqID)
    if err != nil {
        return err
    }

    endF, err := c.receiveEndFrame(reqID)
    if err != nil {
        return err
    }
    if err := c.sendAck(reqID, endFrame); err != nil {
        putFrame(endF)
        return err
    }
    putFrame(endF)

    incoming <- bytes.NewReader(reqBuf.Bytes())
    return nil
}

// Respond() - Receive, process, and respond
func (c *Client) Respond(incoming chan io.Reader, outgoing chan io.Reader) error {
    // Receive message
    if err := c.Receive(incoming); err != nil {
        return err
    }
    
    // Wait for response from application
    reader := <-outgoing
    if reader == nil {
        return nil  // No response to send
    }
    
    // Broadcast response
    return c.Send(reader)
}

func (c *Client) sendAck(reqID uuid.UUID, frameType uint8) error {
    f := getFrame()
    f.Version = version1
    f.Type = ackFrame  // CONTROL FRAME
    f.RequestID = reqID
    f.ClientID = c.clientID
    f.Flags = flagAck  // CONTROL FLAG
    // No payload - control frames don't carry application data
    select {
    case c.ctx.writes <- f:
        return nil
    case <-c.ctx.closed:
        putFrame(f)
        return errClosed
    }
}

func (c *Client) receiveAssembledChunkFramesWithAck(reqID uuid.UUID) (bytes.Buffer, error) {
    var buff bytes.Buffer
    
    for {
        f, err := c.receiveChunkFrame(reqID)
        if err != nil {
            return buff, err
        }
        
        if err := c.sendAck(reqID, chunkFrame); err != nil {
            putFrame(f)
            return buff, err
        }
        
        buff.Write(f.Payload)
        isFinal := f.isFinal()
        putFrame(f)
        
        if isFinal {
            break
        }
    }
    
    return buff, nil
}
```

**Usage Examples**:

```go
// Listen only (no response)
incoming := make(chan io.Reader)
go func() {
    msg := <-incoming
    processMessage(msg)
}()
client.Receive(incoming)

// Listen and respond
incoming := make(chan io.Reader)
outgoing := make(chan io.Reader)
go func() {
    msg := <-incoming
    response := processMessage(msg)
    outgoing <- response
}()
client.Respond(incoming, outgoing)
```

### Implementation Requirements

**SERVER-SIDE CHANGES**:

**registry.go**:
1. Add `pendingAcks map[uuid.UUID]*ackCollector` - track acks per frame
2. Add `ackCollector` struct - manages ack collection and synchronization
3. Add `createAckCollector()` - initialize ack tracking for a frame
4. Add `recordAck()` - record ack from receiver, signal when all received
5. Add `removeAckCollector()` - cleanup after all acks received
6. Add `getChannelReceiverCount()` - count active receivers on channel

**server.go**:
1. Modify flagRequest handler:
   - Check for receivers (getChannelPeers excludes sender)
   - If no receivers: wait with timeout (check every 100ms)
   - On timeout: send error frame "no receivers available"
   - If receivers available: create ack collector
   - Broadcast frame to all receivers
   - Block waiting for all acks
   - Signal sender to proceed after all acks
2. Add flagAck handler:
   - Record ack in registry
   - Trigger ack collector check
3. Keep flagResponse handler (unchanged)
4. Add createErrorFrame() helper:
   - Create error frame with message
   - Set frame type to errorFrame
   - Set flags to flagResponse (control frame)
   - Put error message in payload
   - Copy requestID and other metadata from original frame

**frame.go**:
1. Add `ackFrame (0x07)` type constant - NEW control frame type
2. Add `flagAck (0x02)` flag constant - NEW control frame flag
3. Add `FrameID` field to frame struct (for ack tracking)
4. Mark `flagReceive (0x04)` as deprecated
5. Add helper methods:
   - `isDataFrame()` - returns true for start/chunk/end frames
   - `isControlFrame()` - returns true for ack/error frames
6. Update frame validation to distinguish data vs control frames

**CLIENT-SIDE CHANGES**:

**client.go**:
1. **Modify Send()**:
   - Change signature to `Send(r io.Reader) error` (no response return)
   - After sending each frame (start/chunk/end), call `waitForAck()`
   - Block until ack received from server
2. **Add waitForAck()**:
   - Wait for ack frame from server
   - Handle error frames (e.g., "no receivers available")
   - Extract error message from error frame payload
   - Return error to caller, which aborts Send()
   - Handle timeout
3. **Add Receive()** (new method):
   - Receive broadcast message (server excludes sender)
   - Send acks for each frame
   - Pass message to incoming channel
   - No response broadcasting
4. **Add Respond()** (new method):
   - Call Receive() to get message
   - Wait for response from outgoing channel
   - Broadcast response using Send()
5. **Add sendAck()**:
   - Send ack frame to server
6. **Add receiveAssembledChunkFramesWithAck()**:
   - Receive chunks and send ack for each
7. **Update frame receivers**:
   - Handle ack frames separately from data frames

**TEST UPDATES**:
- Update all tests that expect `Send()` to return response data
- Add tests for ack synchronization
- Test with multiple receivers to verify all get messages
- Test ack timeout scenarios

### Redesign Summary

**New Concepts**:
- **Data frames vs Control frames** - clear separation of concerns
- Ack frames (new control frame type)
- Ack collection and synchronization (server-side)
- Frame-level blocking (sender waits for acks)
- Receiver count tracking per channel
- Server-side sender exclusion from broadcast

**Removed Complexity**:
- flagReceive polling mechanism (deprecated)
- Request-response data pairing
- Response data waiting in Send()
- Client-side self-filtering (server handles exclusion)

**Added Complexity**:
- Ack collection logic (server)
- Ack sending logic (receivers)
- Synchronization between sender and receivers per frame
- FrameID tracking for ack correlation
- Data frame vs control frame handling

### Edge Cases Handled

1. **Multiple concurrent broadcasts**: Each receiver gets all messages from other clients, filters by reqID
2. **Receiver disconnects mid-request**: Ack collection times out or adjusts expected count
3. **No receivers on channel**: 
   - Server waits for receivers with timeout (polls every 100ms)
   - On timeout: server sends error frame with "no receivers available"
   - Client receives error frame in waitForAck()
   - Client's Send() returns error, aborting the send operation
   - Application can retry or handle error appropriately
4. **Concurrent map access**: Existing registry.mu handles server-side synchronization
5. **Ack timeout**: Sender times out if not all acks received within timeout period
6. **Infinite loop prevention**: Use Receive() for listen-only, Respond() for listen-and-respond
7. **Nil response**: Respond() can receive nil from outgoing channel to skip broadcasting
8. **Sender exclusion**: Server's getChannelPeers() excludes sender from peer list
9. **Receivers join after Send() starts**: Server polls for new receivers every 100ms during wait

### API Design Rationale: Separate Receive() and Respond()

**This is the chosen design** - separated methods for clarity and composability.

**Benefits**:
- Clearer intent: `Receive()` for listening only, `Respond()` for request-response pattern
- No need for optional parameters
- Simpler implementation of each method
- More composable
- Prevents accidental infinite loops

### Locking Strategy

**Server-side**:
- Existing `registry.mu` (RWMutex) protects registry maps
- `ackCollector.mu` protects ack counting within each collector
- Blocking on `ackCollector.done` channel for synchronization

**Client-side** (already correct):
- `opMu` on Client prevents concurrent Send()/Receive()
- This is a **business logic constraint**: a client can't be both sender and receiver simultaneously
- Without lock: both operations would read from same `c.ctx.reads` channel and steal each other's frames
- Lock is on Client (not connctx) because:
  - It enforces a logical rule about client roles, not just resource protection
  - connctx is a connection resource; Client is a logical participant
  - The protocol design requires one operation per client at a time

# PulsyFlux - Message Bus System Context

## Project Overview
A reliable and flexible message bus system written in Go that handles message routing between clients through a socket-based protocol.

## Recent Changes to Client.Send()

### Architecture
The `Client.Send()` method has been refactored to assemble response chunks internally before returning an `io.Reader`. Key changes:

1. **Frame Methods Return io.Reader**: 
   - `sendStartFrame()` returns `io.Reader` with start frame response payload
   - `sendChunkFrame()` returns `io.Reader` with chunk frame response payload  
   - `sendEndFrame()` returns `io.Reader` with end frame response payload

2. **Buffer Separation**:
   - Start frame response has its own buffer (NOT shared with chunk/end responses)
   - Chunk and end frame responses share a common `respBuf` buffer
   - `Send()` currently returns only the chunk response buffer (startResp available for future internal reading)

3. **Retry Logic in sendStartFrame()**:
   - Checks for `flagPeerNotAvailable` flag in error frames
   - Retries up to 3 times with 1-second delay between retries
   - Only retries on peer not available errors

4. **Peer ID Management**:
   - If client has no peer ID (`uuid.Nil`), it extracts peer ID from the response start frame's `ClientID` field
   - Logs when peer ID is set

### Frame Protocol
- **Frame Types**: errorFrame (0x03), startFrame (0x04), chunkFrame (0x05), endFrame (0x06)
- **Flags**: 
  - `flagRegistration` (0x01)
  - `flagPeerNotAvailable` (0x02)
- **Frame Structure**: Version, Type, Flags, RequestID, ClientID, PeerClientID, ClientTimeoutMs, Payload

### Flow
1. Send START frame → receive START response → extract peer ID if needed
2. Loop: Send CHUNK frames → receive CHUNK responses → accumulate in respBuf
3. Send END frame → receive END response → accumulate in respBuf
4. Return combined response as io.Reader

## Key Components

### Client Structure
```go
type Client struct {
    addr      string
    ctx       *connctx
    connMu    sync.Mutex
    clientID  uuid.UUID
    peerID    uuid.UUID
    channelID uuid.UUID
    role      clientRole
    done      chan struct{}
}
```

### Connection Context
- Manages TCP connection with frame-based protocol
- Separate channels for writes, reads, and errors (priority)
- Frame header size: 64 bytes
- Max frame size: 1MB

## TODO/Future Work
- Internal reading of `startResp` in `Send()` method (currently unused)
- Consider timeout handling for retry logic
- Error handling improvements

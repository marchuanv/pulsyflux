# PulsyFlux - Message Bus System Context

## Project Overview
A reliable and flexible message bus system written in Go that handles message routing between clients through a socket-based protocol.

## Recent Changes to Client Architecture

### Client.Send() Refactoring
The `Client.Send()` method has been refactored to assemble response chunks internally before returning an `io.Reader`:

1. **Frame Methods Return io.Reader**: 
   - `sendStartFrame()` now returns `error` only (no longer returns io.Reader)
   - `sendChunkFrame()` returns `io.Reader` with chunk frame response payload  
   - `sendEndFrame()` returns `io.Reader` with end frame response payload

2. **Buffer Management**:
   - Start frame response is handled internally (no buffer returned)
   - Chunk and end frame responses share a common `respBuf` buffer
   - `Send()` returns only the chunk/end response buffer

3. **Retry Logic in sendStartFrame()**:
   - Checks for `flagPeerNotAvailable` flag in error frames
   - Retries up to 3 times with 1-second delay between retries
   - Uses `goto retryLoop` to properly break out of inner select loop
   - Only retries on peer not available errors

4. **Peer ID and Channel ID Management**:
   - If client has no peer ID (`uuid.Nil`), it extracts peer ID from response start frame's `ClientID` field
   - Channel ID is read from first 16 bytes of start frame response payload
   - Peer ID is only set if channel IDs match
   - Returns error if peer ID is set but response ClientID doesn't match

5. **Role Validation**:
   - Start frame payload contains: channelID (16 bytes) + role (1 byte)
   - Validates that peer role is different from client role
   - Returns error if roles are the same

6. **Registration Flag**:
   - `flagRegistration` is set on start frame when `peerID == uuid.Nil`
   - Indicates a registration/handshake request

### Handshake Mechanism
- Automatic handshake in `NewClient()` constructor
- Runs in separate goroutine with proper WaitGroup management (count: 3)
- **Timing delays to prevent simultaneous handshakes**:
  - Consumer clients: 50ms delay before handshake
  - Provider clients: 150ms delay before handshake
  - Ensures clients don't send registration frames at exactly the same time

### Frame Protocol
- **Frame Types**: errorFrame (0x03), startFrame (0x04), chunkFrame (0x05), endFrame (0x06)
- **Flags**: 
  - `flagRegistration` (0x01) - indicates registration request
  - `flagPeerNotAvailable` (0x02) - no peer available for channel
- **Frame Structure**: Version, Type, Flags, RequestID, ClientID, PeerClientID, ClientTimeoutMs, Payload
- **Start Frame Payload**: channelID (16 bytes) + role (1 byte)

### Flow
1. **Handshake**: Send START frame with registration flag → receive START response → extract peer ID and validate channel/role
2. **Request**: Send START frame → receive START response
3. Loop: Send CHUNK frames → receive CHUNK responses → accumulate in respBuf
4. Send END frame → receive END response → accumulate in respBuf
5. Return combined response as io.Reader

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

### Client Roles
- `roleConsumer` (0x01) - sends requests
- `roleProvider` (0x02) - handles requests
- Roles must be different for peer pairing

### Connection Context
- Manages TCP connection with frame-based protocol
- Separate channels for writes, reads, and errors (priority)
- Frame header size: 64 bytes
- Max frame size: 1MB
- WaitGroup tracks 3 goroutines: writer, reader, handshake

### Server Handling
- Registers clients on `flagRegistration` start frames
- Routes frames between paired clients based on channel ID
- Returns `flagPeerNotAvailable` error when no peer exists

## Error Handling
- `errPeerError` - peer-related errors (mismatch, role conflict, etc.)
- `errClosed` - client/connection closed
- Peer ID mismatch detection
- Role conflict detection
- Channel ID validation

## Testing Considerations
- Handshake timing delays prevent race conditions
- Multiple consumers can connect to single provider
- Concurrent channels supported
- Large payload handling (2MB+)
- No peer available scenarios handled with retries

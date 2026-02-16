# Change Request: Server-Side Multiplexing Support

## Problem

The broker package requires bidirectional multiplexed communication where:
1. Multiple clients connect to the same server
2. Server needs to send messages to specific clients using their UUIDs
3. Server-side `WrapConnection` currently only supports sending with its own UUID

**Current Limitation:**
- `WrapConnection` can only send with the UUID it was created with
- Server wraps accepted connections with empty UUID
- When server sends, all messages have empty UUID
- Client-side demux cannot route empty UUID messages to specific clients

## Current Behavior

```go
// Server side
conn, _ := listener.Accept()
wrapped := tcpconn.WrapConnection(conn, uuid.UUID{}) // Empty UUID

// Server can only send with empty UUID
wrapped.Send(data) // Sends with empty UUID, not client's UUID
```

## Requested Change

Add ability for wrapped connections to send with a target UUID:

```go
// Option 1: Add SendTo method
func (c *Connection) SendTo(data []byte, targetID uuid.UUID) error

// Option 2: Modify Send to accept optional UUID
func (c *Connection) Send(data []byte, targetID ...uuid.UUID) error

// Option 3: Add SetTargetID method
func (c *Connection) SetTargetID(id uuid.UUID)
func (c *Connection) Send(data []byte) error // Uses target ID
```

## Use Case

```go
// Server receives message from client
clientID, _ := uuid.Parse(clientMsg.ClientID)

// Server needs to send response to specific client
wrapped.SendTo(responseData, clientID) // Routes to correct client
```

## Alternative Solution

If modifying Send API is not desired, consider:

**Server-side demux for wrapped connections:**
- Allow multiple `WrapConnection` calls on same `net.Conn`
- Add server-side demux similar to client-side pool demux
- Each wrapped connection registers its UUID with a wrapper pool

```go
// Multiple wrapped connections on same accepted socket
conn, _ := listener.Accept()
client1 := tcpconn.WrapConnection(conn, clientID1) // Registers with wrapper pool
client2 := tcpconn.WrapConnection(conn, clientID2) // Shares same demux

// Each can send/receive independently
client1.Send(data) // Tagged with clientID1
client2.Receive()  // Receives only clientID2 messages
```

## Impact

**Without this change:**
- Broker cannot properly route messages to specific clients
- Senders receive their own messages (broadcast issue)
- Server-side multiplexing is not possible

**With this change:**
- Server can send targeted messages to specific clients
- Broker can implement proper pub/sub with sender exclusion
- Full bidirectional multiplexing support

## Priority

**High** - Blocks broker implementation and any server-side multiplexing use cases

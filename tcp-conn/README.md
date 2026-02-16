# TCP-Conn

A minimal, high-performance TCP connection abstraction with automatic lifecycle management, connection pooling, multiplexing, and reconnection support for both client and server applications.

## Features

- **Global Connection Pool**: Multiple logical connections automatically share physical TCP connections
- **Connection Multiplexing**: Multiple logical connections over a single TCP socket with ID-based routing
- **Auto-Reconnect**: Client connections automatically reconnect on disconnect
- **Idle Timeout**: Automatic cleanup of inactive connections
- **Thread-Safe**: All operations protected with mutexes
- **Client & Server**: Works on both sides with appropriate behavior
- **Zero Configuration**: Sensible defaults, minimal API

## Public API

```go
type Connection struct {
    // internal fields
}

// Client-side: Creates connection with auto-reconnect
func NewConnection(address string, idleTimeout time.Duration) *Connection
func NewConnectionWithID(address string, id string, idleTimeout time.Duration) *Connection

// Server-side: Wraps accepted connection (no reconnect)
func WrapConnection(conn net.Conn, idleTimeout time.Duration) *Connection
func WrapConnectionWithID(conn net.Conn, id string, idleTimeout time.Duration) *Connection

// Send/Receive operations
func (c *Connection) Send(data []byte) error
func (c *Connection) Receive() ([]byte, error)
```

## Usage

### Basic Client Connection

```go
import tcpconn "github.com/pulsyflux/tcp-conn"

// Simple connection with defaults
c := tcpconn.NewConnection("localhost:8080", 5*time.Minute)

c.Send([]byte("hello"))
data, _ := c.Receive()
```

### Connection Multiplexing

Multiple logical connections share the same physical TCP connection:

```go
// Both connections share the same underlying TCP socket
conn1 := tcpconn.NewConnectionWithID("localhost:8080", "user-session", 5*time.Minute)
conn2 := tcpconn.NewConnectionWithID("localhost:8080", "admin-session", 5*time.Minute)

// Messages are routed by ID
conn1.Send([]byte("user data"))    // Tagged with "user-session"
conn2.Send([]byte("admin data"))   // Tagged with "admin-session"

// Each receives only its own messages
userData, _ := conn1.Receive()     // Gets "user data" only
adminData, _ := conn2.Receive()    // Gets "admin data" only
```

### Server-Side Usage

```go
listener, _ := net.Listen("tcp", ":8080")

for {
    conn, _ := listener.Accept()
    
    // Wrap accepted connection
    c := tcpconn.WrapConnection(conn, 5*time.Minute)
    
    go func() {
        for {
            data, err := c.Receive()
            if err != nil {
                return // Client disconnected or idle timeout
            }
            c.Send(data) // Echo back
        }
    }()
}
```

### Server with Multiple Logical Connections

```go
listener, _ := net.Listen("tcp", ":8080")

for {
    conn, _ := listener.Accept()
    
    // Create multiple logical connections over same accepted socket
    session1 := tcpconn.WrapConnectionWithID(conn, "session-1", 5*time.Minute)
    session2 := tcpconn.WrapConnectionWithID(conn, "session-2", 5*time.Minute)
    
    go handleSession(session1)
    go handleSession(session2)
}
```

## Architecture

### Connection Pooling

```
Client Side:
┌─────────────────────────────────────────┐
│         Global Connection Pool          │
│  ┌────────────────────────────────────┐ │
│  │ "server:8080" → TCP Connection     │ │
│  │   RefCount: 3                      │ │
│  └────────────────────────────────────┘ │
└─────────────────────────────────────────┘
           ↑         ↑         ↑
           │         │         │
    ┌──────┴──┐ ┌───┴────┐ ┌──┴──────┐
    │ Conn 1  │ │ Conn 2 │ │ Conn 3  │
    │ ID: "A" │ │ ID: "B"│ │ ID: "C" │
    └─────────┘ └────────┘ └─────────┘
```

### Message Framing

All messages are framed with connection ID for multiplexing:

```
┌──────────┬────────┬─────────────┬──────────┐
│ ID Length│   ID   │ Data Length │   Data   │
│  (1 byte)│(N bytes)│  (4 bytes)  │(M bytes) │
└──────────┴────────┴─────────────┴──────────┘
```

- **ID Length**: 1 byte indicating ID string length
- **ID**: Variable length connection identifier
- **Data Length**: 4 bytes (uint32) indicating payload size
- **Data**: Actual payload bytes

### Lifecycle Management

**Client-Side:**
1. Connection created → Added to global pool
2. Idle timeout → Connection closed, removed from pool
3. Disconnect → Auto-reconnect on next Send/Receive
4. Multiple logical connections → Share same physical connection

**Server-Side:**
1. Accept connection → Wrap with WrapConnection
2. Idle timeout → Connection closed
3. Disconnect → No reconnect (client's responsibility)
4. Each accepted connection is independent

## Configuration

### Idle Timeout

Default: 5 minutes. Connections close after inactivity.

```go
// Custom timeout
c := tcpconn.NewConnection("localhost:8080", 10*time.Minute)

// Use default (5 minutes)
c := tcpconn.NewConnection("localhost:8080", 0)
```

### Connection IDs

- **Empty ID** (`""`): Accepts all messages (default)
- **Specific ID**: Only receives messages tagged with that ID

```go
// No filtering - receives all messages
c1 := tcpconn.NewConnection("localhost:8080", 5*time.Minute)

// Filtered - only receives messages for "session-123"
c2 := tcpconn.NewConnectionWithID("localhost:8080", "session-123", 5*time.Minute)
```

## Error Handling

```go
err := conn.Send([]byte("data"))
if err != nil {
    // Connection closed, reconnect failed, or network error
}

data, err := conn.Receive()
if err != nil {
    // Connection closed, reconnect failed, or network error
}
```

## Performance Characteristics

- **Connection Pooling**: Reduces TCP handshake overhead
- **Multiplexing**: Multiple logical connections without multiple sockets
- **Zero-Copy**: Minimal memory allocations
- **Thread-Safe**: Lock-free reads where possible
- **Auto-Cleanup**: Reference counting prevents resource leaks

## Design Philosophy

1. **Minimal API**: Only Send, Receive, and constructors
2. **Auto-Management**: No manual state management required
3. **Global Pool**: Automatic connection sharing
4. **Transparent Multiplexing**: ID-based routing without user intervention
5. **Error-Based State**: Check errors, not state methods

## Thread Safety

All operations are thread-safe:
- Multiple goroutines can call Send/Receive on the same connection
- Global pool is protected with RWMutex
- Reference counting is atomic

## Limitations

- **Read Buffer**: 4KB default (sufficient for most use cases)
- **Blocking I/O**: Send/Receive block until complete
- **No Partial Reads**: Messages must fit in memory
- **TCP Only**: No UDP support

## Best Practices

1. **Use Connection IDs** for multiplexing multiple logical connections
2. **Set appropriate idle timeouts** based on your use case
3. **Handle errors** - they indicate connection state
4. **Server-side**: One WrapConnection per accepted connection
5. **Client-side**: Reuse connections to same address for pooling benefits

## Examples

### Request-Response Pattern

```go
// Client
client := tcpconn.NewConnection("server:8080", 5*time.Minute)
client.Send([]byte("REQUEST"))
response, _ := client.Receive()

// Server
listener, _ := net.Listen("tcp", ":8080")
conn, _ := listener.Accept()
wrapped := tcpconn.WrapConnection(conn, 5*time.Minute)
request, _ := wrapped.Receive()
wrapped.Send([]byte("RESPONSE"))
```

### Bidirectional Streaming

```go
// Both sides can send/receive concurrently
go func() {
    for {
        data, _ := conn.Receive()
        process(data)
    }
}()

go func() {
    for {
        conn.Send(generateData())
        time.Sleep(time.Second)
    }
}()
```

### Connection Pool Sharing

```go
// All three share the same TCP connection
conn1 := tcpconn.NewConnectionWithID("api:8080", "user-1", 5*time.Minute)
conn2 := tcpconn.NewConnectionWithID("api:8080", "user-2", 5*time.Minute)
conn3 := tcpconn.NewConnectionWithID("api:8080", "user-3", 5*time.Minute)

// Efficient: Only one TCP connection to api:8080
// Messages routed by ID automatically
```

## License

See LICENSE file for details.

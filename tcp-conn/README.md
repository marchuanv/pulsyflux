# TCP-Conn

A minimal TCP connection abstraction with automatic lifecycle management, reconnection, and performance scaling. Works for both client and server side.

## Public API

```go
type Connection struct {
    // internal fields
}

func (c *Connection) Send(data []byte) error
func (c *Connection) Receive() ([]byte, error)

// Client-side: creates and manages connection
func NewConnection(address string, idleTimeout time.Duration) *Connection

// Server-side: wraps accepted connection
func WrapConnection(conn net.Conn, idleTimeout time.Duration) *Connection
```

## Usage

### Client-Side

```go
import tcpconn "github.com/pulsyflux/tcp-conn"

// Connection is created and managed internally
c := tcpconn.NewConnection("localhost:8080", 5*time.Minute)

c.Send([]byte("hello"))
data, _ := c.Receive()

// Automatically reconnects after idle timeout or disconnect
```

### Server-Side

```go
listener, _ := net.Listen("tcp", ":8080")

for {
    conn, _ := listener.Accept()
    
    // Wrap accepted connection
    c := tcpconn.WrapConnection(conn, 5*time.Minute)
    
    go func() {
        data, _ := c.Receive()
        c.Send(data) // Echo back
    }()
}
```

## Features

- **Client-side**: Auto-dial, auto-reconnect, connection pooling ready
- **Server-side**: Wrap accepted connections with idle timeout
- **Idle timeout**: Connections close after inactivity (default: 5 minutes)
- **Thread-safe**: All operations protected with mutexes
- **Minimal API**: Only Send and Receive, works directly with bytes

## Design

**Client-side** (`NewConnection`):
- Creates connections internally
- Automatically reconnects on disconnect
- Ready for connection pooling

**Server-side** (`WrapConnection`):
- Wraps accepted `net.Conn`
- Idle timeout management
- No reconnect (server doesn't reconnect to clients)

Both provide the same `Send`/`Receive` interface for consistent usage.

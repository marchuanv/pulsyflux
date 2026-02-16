# TCP-Conn

A minimal TCP connection abstraction with automatic lifecycle management and reconnection.

## Public API

```go
type Connection struct {
    // internal fields
}

func (c *Connection) Send(data []byte) error
func (c *Connection) Receive() ([]byte, error)

func NewConnection(
    conn net.Conn,
    idleTimeout time.Duration,
    reconnectFn func() (net.Conn, error),
) *Connection
```

## Usage

### Basic Connection

```go
import tcpconn "github.com/pulsyflux/tcp-conn"

conn, _ := net.Dial("tcp", "localhost:8080")
c := tcpconn.NewConnection(conn, 5*time.Minute, nil)

c.Send([]byte("hello"))
data, _ := c.Receive()
```

### Connection with Auto-Reconnect

```go
reconnectFn := func() (net.Conn, error) {
    return net.Dial("tcp", "localhost:8080")
}

conn, _ := net.Dial("tcp", "localhost:8080")
c := tcpconn.NewConnection(conn, 5*time.Minute, reconnectFn)

// Automatically reconnects on Send/Receive if disconnected
c.Send([]byte("hello"))
data, _ := c.Receive()
```

## Features

- **Auto-reconnect**: Automatically re-establishes connection on Send/Receive
- **Idle timeout**: Connections close after inactivity (default: 5 minutes)
- **Thread-safe**: All operations protected with mutexes
- **Minimal API**: Only Send and Receive, works directly with bytes

## Design

State management is completely internal. Users only call Send/Receive and handle errors. Connections automatically reconnect when needed.

If you need message framing, implement it in your application layer using the raw Send/Receive methods.

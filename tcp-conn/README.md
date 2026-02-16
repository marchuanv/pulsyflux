# TCP-Conn

A minimal TCP connection abstraction with automatic lifecycle management, reconnection, and performance scaling.

## Public API

```go
type Connection struct {
    // internal fields
}

func (c *Connection) Send(data []byte) error
func (c *Connection) Receive() ([]byte, error)

func NewConnection(address string, idleTimeout time.Duration) *Connection
```

## Usage

### Basic Connection

```go
import tcpconn "github.com/pulsyflux/tcp-conn"

// Connection is created and managed internally
c := tcpconn.NewConnection("localhost:8080", 5*time.Minute)

c.Send([]byte("hello"))
data, _ := c.Receive()
```

### Auto-Reconnect

```go
c := tcpconn.NewConnection("localhost:8080", 5*time.Minute)

// Automatically reconnects on Send/Receive if disconnected
c.Send([]byte("hello"))
data, _ := c.Receive()

// After idle timeout, next Send/Receive will reconnect
time.Sleep(6 * time.Minute)
c.Send([]byte("reconnected")) // Automatically reconnects
```

## Features

- **Internal connection management**: No need to pass net.Conn, just provide address
- **Auto-reconnect**: Automatically re-establishes connection on Send/Receive
- **Idle timeout**: Connections close after inactivity (default: 5 minutes)
- **Performance scaling**: Connection pool ready for high-throughput scenarios
- **Thread-safe**: All operations protected with mutexes
- **Minimal API**: Only Send and Receive, works directly with bytes

## Design

Connections are created and managed internally. Users only provide the address and call Send/Receive. The connection automatically:
- Establishes on first use
- Reconnects when disconnected
- Closes after idle timeout
- Scales with connection pooling for performance

No manual connection management needed.

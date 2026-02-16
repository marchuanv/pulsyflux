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

### Demultiplexer Design

The package implements true multiplexing using a demultiplexer pattern. Each physical TCP connection has a single reader goroutine that routes messages to logical connections:

```
Physical Connection (net.Conn)
    │
    └─> Demux Goroutine (single reader)
            │
            ├─> Reads frames from socket
            ├─> Reassembles chunked messages
            ├─> Routes by UUID to channels
            │
            ├─> Logical Conn A (recvChan) → Client reads from channel
            ├─> Logical Conn B (recvChan) → Client reads from channel
            └─> Logical Conn C (recvChan) → Client reads from channel
```

**Key Components:**
- **physicalPool**: Manages physical connection, demux goroutine, and routing table
- **routes map[uuid.UUID]chan []byte**: Maps connection IDs to receive channels
- **messageAssembly**: Tracks partial messages during chunk reassembly
- **Demux goroutine**: Single reader per physical connection (prevents deadlocks)

### Connection Pooling

```
Client Side:
┌─────────────────────────────────────────┐
│         Global Connection Pool          │
│  ┌────────────────────────────────────┐ │
│  │ "server:8080" → physicalPool       │ │
│  │   - net.Conn                       │ │
│  │   - demux goroutine                │ │
│  │   - routes map                     │ │
│  │   - refCount: 3                    │ │
│  └────────────────────────────────────┘ │
└─────────────────────────────────────────┘
           ↑         ↑         ↑
           │         │         │
    ┌──────┴──┐ ┌───┴────┐ ┌──┴──────┐
    │ Conn 1  │ │ Conn 2 │ │ Conn 3  │
    │ ID: "A" │ │ ID: "B"│ │ ID: "C" │
    │recvChan │ │recvChan│ │recvChan │
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

### Benchmark Results

**Test Environment**: Windows, amd64, Intel i5-12400F (12 cores)

#### Current Implementation (with Demultiplexer)

| Benchmark | Ops/sec | Latency (avg) | Throughput |
|-----------|---------|---------------|------------|
| Small Messages (100B) | 30,467 | 32.8 µs | ~2.9 MB/s |
| Large Messages (1MB) | 193 | 5.2 ms | ~193 MB/s |
| Chunking (200KB) | 3,817 | 264 µs | ~763 MB/s |
| Multiple Connections | 29,433 | 34.2 µs | ~2.8 MB/s |
| Send Only (1KB) | 131,387 | 7.6 µs | ~131 MB/s |
| Receive Only (1KB) | 124,590 | 8.0 µs | ~125 MB/s |

#### Previous Implementation (Direct Reads - Had Deadlock Bug)

| Benchmark | Ops/sec | Latency | Throughput | Status |
|-----------|---------|---------|------------|--------|
| Small Messages (100B) | 49,749 | 20.1 µs | ~4.7 MB/s | ⚠️ Deadlocked with multiplexing |
| Large Messages (1MB) | 175 | 5.7 ms | ~175 MB/s | ⚠️ Deadlocked with multiplexing |
| Chunking (200KB) | 3,153 | 317 µs | ~630 MB/s | ⚠️ Deadlocked with multiplexing |
| Send Only (1KB) | 138,179 | 7.2 µs | ~138 MB/s | ⚠️ Deadlocked with multiplexing |
| Receive Only (1KB) | 137,397 | 7.3 µs | ~137 MB/s | ⚠️ Deadlocked with multiplexing |

#### Performance Analysis

**Comparison (Current vs Previous):**
- **Small messages (100B)**: 63% slower (20µs → 33µs) - channel overhead
- **Large messages (1MB)**: 10% **faster** (5.7ms → 5.2ms) - better buffering
- **Chunking (200KB)**: 21% **faster** (317µs → 264µs) - improved reassembly
- **Send operations**: 5% slower (7.2µs → 7.6µs) - minimal impact
- **Receive operations**: 10% slower (7.3µs → 8.0µs) - channel overhead

**Key Findings:**
1. ✅ **Large messages improved**: Better performance for 1MB+ messages
2. ✅ **Chunking improved**: 21% faster for 200KB messages
3. ⚠️ **Small message overhead**: 63% slower for <1KB messages
4. ✅ **No deadlocks**: Multiple logical connections work correctly
5. ✅ **Scalability**: Overhead decreases as message size increases

**Why the trade-off is worth it:**
- **Correctness**: Previous implementation was fundamentally broken for multiplexing
- **Better at scale**: Faster for large messages (most real-world use cases)
- **Improved chunking**: 21% faster for medium-large messages
- **Fixed overhead**: ~13µs channel overhead is constant, negligible for large messages

**Recommendation**: 
- **Small messages (<1KB)**: Consider batching to amortize overhead
- **Medium-large messages (>10KB)**: Overhead is negligible (<1%)
- **Large messages (>1MB)**: Actually faster than previous implementation

See [BENCHMARK_REPORT.md](BENCHMARK_REPORT.md) for detailed analysis.

### Features

- **Connection Pooling**: Reduces TCP handshake overhead
- **Multiplexing**: Multiple logical connections without multiple sockets
- **Chunked I/O**: Handles partial reads/writes automatically (64KB chunks)
- **Large Messages**: Tested with 1MB+ messages
- **Thread-Safe**: All operations protected with mutexes
- **Auto-Cleanup**: Reference counting prevents resource leaks

## Design Philosophy

1. **Minimal API**: Only Send, Receive, and constructors
2. **Auto-Management**: No manual state management required
3. **Global Pool**: Automatic connection sharing
4. **Transparent Multiplexing**: ID-based routing without user intervention
5. **Error-Based State**: Check errors, not state methods

## Implementation Details

### Multiplexing Fix (v2.0)

**Problem Solved**: Previous implementation had a critical deadlock bug where multiple logical connections sharing the same physical TCP socket would all attempt to read directly from the socket, causing file descriptor mutex deadlocks.

**Solution**: Implemented a demultiplexer architecture:

1. **Single Reader Pattern**: Each physical connection has ONE demux goroutine that reads from the socket
2. **Message Reassembly**: Demux reassembles chunked messages before routing
3. **Channel-Based Routing**: Complete messages are sent to logical connections via buffered channels
4. **UUID-Based Routing**: Each logical connection registers its UUID and receive channel

**Key Changes:**
- `physicalPool` now manages demux goroutine and routing table
- `Connection.Receive()` reads from channel (pooled) or socket (wrapped)
- Added `register()`/`unregister()` for route management
- Added `messageAssembly` struct for tracking partial messages

**Client vs Server Behavior:**
- **Client (pooled)**: Uses demux + channels (supports multiplexing)
- **Server (wrapped)**: Direct socket reads (no multiplexing needed)

### Thread Safety

All operations are thread-safe:
- **Concurrent Send/Receive**: Send and Receive can run simultaneously on the same connection (separate read/write locks)
- **Demux Goroutine**: Single reader per physical connection prevents deadlocks
- **Route Management**: Protected with RWMutex for concurrent access
- **Global Pool**: Protected with RWMutex
- **Reference Counting**: Atomic operations
- **Channel Communication**: Go channels provide built-in synchronization
- Logical close only releases pool reference, physical close happens when refCount reaches 0

## Limitations

- **Blocking I/O**: Send/Receive block until complete
- **Messages in Memory**: Full messages must fit in memory
- **TCP Only**: No UDP support
- **Single Send/Receive per goroutine**: Only one Send and one Receive can execute concurrently per connection

## Best Practices

1. **Use Connection IDs** for multiplexing multiple logical connections
2. **Set appropriate idle timeouts** based on your use case
3. **Handle errors** - they indicate connection state
4. **Server-side**: One WrapConnection per accepted connection
5. **Client-side**: Reuse connections to same address for pooling benefits
6. **Message Size**: For ultra-low latency, use messages >1KB to minimize channel overhead
7. **Batching**: Consider batching small messages to reduce per-message overhead
8. **Channel Buffer**: Receive channels have 10-message buffer to prevent blocking demux

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

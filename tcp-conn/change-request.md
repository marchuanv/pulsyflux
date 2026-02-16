# Change Request: True Multiplexing Support

## Problem

The tcp-conn package claims to support multiplexing but does not actually implement it correctly. Multiple logical connections sharing the same physical TCP socket all attempt to read directly from the socket, causing file descriptor mutex deadlocks.

### Current Broken Behavior

```go
// Client side - DEADLOCK
control := tcpconn.NewConnection("server:8080", controlUUID)  // Uses pool
channel := tcpconn.NewConnection("server:8080", channelUUID)  // Reuses same physical conn

// Both call globalPool.get("server:8080") → same net.Conn
// Both spawn goroutines calling conn.Read() → DEADLOCK on fd mutex

// Server side - DEADLOCK  
conn, _ := listener.Accept()
control := tcpconn.WrapConnection(conn, controlUUID)
channel := tcpconn.WrapConnection(conn, channelUUID)

// Both try to read from same net.Conn → DEADLOCK
```

### Stack Trace Evidence

```
goroutine 21 [semacquire]:
internal/poll.(*fdMutex).rwlock(0xc00008e008, 0x0?)
internal/poll.(*FD).Read(0xc00008e008, {0xc00008a059, 0x1, 0x1})
net.(*conn).Read(0xc00009e000, ...)
pulsyflux/tcp-conn.(*Connection).readFull(...)
pulsyflux/tcp-conn.(*Connection).Receive(...)

goroutine 23 [IO wait]:
internal/poll.(*FD).Read(0xc00008e008, {0xc00008a058, 0x1, 0x1})
net.(*conn).Read(0xc00009e000, ...)  // SAME net.Conn!
pulsyflux/tcp-conn.(*Connection).readFull(...)
```

Both goroutines blocked trying to read from `0xc00009e000` (same physical connection).

## Root Cause

Each logical `Connection` calls `readFull()` which directly reads from the physical `net.Conn`. When multiple logical connections share a physical connection via the pool, they all try to read simultaneously from the same file descriptor, causing the OS-level mutex deadlock.

**The package does NOT have a demultiplexer.**

## Required Changes

### Architecture: Add Demultiplexer

Each physical connection needs ONE reader goroutine that:
1. Reads frames from the socket
2. Parses the connection ID from each frame
3. Routes the frame data to the appropriate logical connection via a channel

```
Physical Connection (net.Conn)
       │
       ├─> Demux Goroutine (single reader)
       │        │
       │        ├─> routes to Logical Conn A (via channel)
       │        ├─> routes to Logical Conn B (via channel)
       │        └─> routes to Logical Conn C (via channel)
       │
Logical Connections (receive from channels, no direct reads)
```

### Implementation Plan

1. **Add demux goroutine to physicalPool**
   ```go
   type physicalPool struct {
       address  string
       conn     net.Conn
       mu       sync.RWMutex
       refCount int32
       routes   map[string]chan []byte  // UUID -> data channel
       routesMu sync.RWMutex
   }
   ```

2. **Start demux on first connection**
   ```go
   func (pp *physicalPool) startDemux() {
       go func() {
           for {
               // Read frame: [id_len][id][total_len][chunk_len][chunk]
               frame := readFrameFromConn(pp.conn)
               
               pp.routesMu.RLock()
               ch := pp.routes[frame.id]
               pp.routesMu.RUnlock()
               
               if ch != nil {
                   ch <- frame.data
               }
           }
       }()
   }
   ```

3. **Logical Connection receives from channel**
   ```go
   func (t *Connection) Receive() ([]byte, error) {
       // Don't read from conn directly
       // Read from demux channel instead
       data := <-t.recvChan
       return data, nil
   }
   ```

4. **Register/unregister routes**
   ```go
   func (cp *connectionPool) register(address string, id uuid.UUID, ch chan []byte)
   func (cp *connectionPool) unregister(address string, id uuid.UUID)
   ```

### Breaking Changes

- `Connection.Receive()` behavior changes from blocking read to channel receive
- Pool must track logical connection registrations
- Demux goroutine lifecycle management needed

## Workaround (Current)

Until this is fixed, applications must ensure:
- **Only ONE logical connection per physical connection**
- **No connection pooling when multiplexing is needed**
- **Each logical connection gets its own TCP socket**

This defeats the purpose of multiplexing but avoids deadlocks.

## Impact

- **Broker package**: Currently deadlocks when using control + channel connections
- **Any application**: Cannot safely create multiple logical connections to same address
- **Performance**: Connection pooling benefit is lost

## Priority

**HIGH** - Current implementation is fundamentally broken for the advertised use case.

## Concurrent Send/Receive Support

**COMPLETED** - Changed from single `inUse` lock to separate `readInUse` and `writeInUse` locks, allowing concurrent Send and Receive operations on the same logical connection. This is required for bidirectional control connections where both client and server need to send/receive simultaneously.

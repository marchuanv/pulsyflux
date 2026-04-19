---
inclusion: auto
---

# PulsyFlux Coding Standards

## Go Conventions

- Package names: lowercase, single word (`broker`, `tcpconn`)
- Internal tcp-conn import path: `pulsyflux/tcp-conn` aliased as `tcpconn`
- Use `sync.RWMutex` for concurrent map access, `atomic` for counters and flags
- Error handling: return errors, don't panic. Log and continue for non-fatal connection errors.
- Use `uuid.UUID` from `github.com/google/uuid` for all identifiers
- Channel buffers: use appropriate capacity (10 for demux recv channels, 100 for subscriber channels)
- Use `context.Context` for goroutine lifecycle management
- Goroutine cleanup: use `done` channels or context cancellation
- Test files: `*_test.go` in same package, benchmark files: `*_bench_test.go`

## Message Framing Protocol

All tcp-conn messages use this wire format:
```
[ID Length: 1 byte][ID: N bytes (UUID string)][Total Length: 4 bytes big-endian][Chunk Length: 4 bytes big-endian][Data: M bytes]
```
- Chunking at 64KB boundaries for large messages
- Demuxer reassembles chunks before routing to logical connection

## Broker Control Protocol

- GlobalControlUUID: `00000000-0000-0000-0000-000000000000`
- Control message format: JSON `{"client_id": "uuid", "channel_id": "uuid"}`
- Handshake: client sends control msg → server registers → server sends ack byte (0x01) → client proceeds

## Node.js Addon Conventions

- C++ addon uses N-API (node-addon-api)
- Go shared library exports use cgo with C-compatible signatures
- Memory: Go allocates via `C.CBytes`, Node.js copies via `Napi::Buffer::Copy`, caller frees via `FreePayload`
- Async message receiving: `MessageWorker` (AsyncWorker) polls with 1ms sleep, 100 iterations per cycle
- ES module wrapper in `registry.mjs` — keep the JS API clean and minimal

## Thread Safety Patterns

- One `Send` and one `Receive` per connection at a time (atomic CAS guards)
- Single demux reader goroutine per physical connection — never read from socket directly
- Pool access: RWMutex (read-heavy, write-rare)
- Channel broadcast: RLock on channel clients map during iteration
- Server channels map: Lock for writes, RLock for reads

## Testing Patterns

- Go: table-driven tests, use `time.After` for timeout assertions
- Go benchmarks: use `b.N` loop, report `-benchmem`
- Node.js: Jasmine with `done` callback for async tests, `setTimeout` for connection setup delays
- Always use `:0` for server address in tests (OS-assigned port)

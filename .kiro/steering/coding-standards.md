---
inclusion: auto
---

# Coding Standards

## Go Conventions

- Package names: lowercase, single word (`broker`)
- tcp import: `github.com/pulsyflux/tcp` aliased as `tcpconn`
- Use `sync.RWMutex` for concurrent map access
- Error handling: return errors, don't panic. Log and continue for non-fatal connection errors.
- Use `uuid.UUID` from `github.com/google/uuid` for all identifiers
- Subscriber channel buffer capacity: 100
- Goroutine cleanup: use `done` channels for shutdown signaling
- Test files: `*_test.go` in same package, benchmark files: `*_bench_test.go`

## Thread Safety Patterns

- Channel broadcast: RLock on channel clients map during iteration
- Server channels map: Lock for writes, RLock for reads
- Client subs slice: Lock for writes (Subscribe), RLock for reads (receiveLoop)

## Testing Patterns

- Use `time.After` for timeout assertions in select blocks
- Go benchmarks: use `b.N` loop, report `-benchmem`
- Always use `:0` for server address in tests (OS-assigned port)
- Use `time.Sleep` for connection setup delays in tests

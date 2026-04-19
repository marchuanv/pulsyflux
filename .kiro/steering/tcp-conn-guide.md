---
inclusion: fileMatch
fileMatchPattern: "tcp-conn/**"
---

# tcp-conn Package Guide

## Key Files

- `logical.go` — `Connection` struct, `NewConnection`, `WrapConnection`, `Send`, `Receive`, idle monitor
- `pool.go` — `connectionPool` (global), `physicalPool` (client-side), `wrappedPool` (server-side)
- `demux.go` — `demuxer` struct, single-reader goroutine, UUID-based message routing, chunk reassembly
- `errors.go` — sentinel errors: `errConnectionClosed`, `errConnectionDead`, `errConnectionInUse`

## Connection Lifecycle

1. Client: `NewConnection(address, id)` → pool `getOrCreate` → register route in demuxer → start idle monitor
2. Server: `WrapConnection(conn, id)` → `getOrCreateWrappedPool` → register route → start idle monitor
3. Idle timeout (30s) → `close()` → unregister from demuxer → release pool ref → physical close at refCount 0

## Send Frame Format

```
[idLen:1][id:N][totalLen:4][chunkLen:4][data:M]
```
- Chunks at 64KB, each chunk gets full header
- Big-endian uint32 for lengths

## Demuxer

- One goroutine per physical connection reads frames continuously
- Routes complete messages to registered UUID channels
- `messageAssembly` tracks partial messages across chunks
- Buffered channels (cap 10) prevent demux blocking

## Important Constraints

- Only one concurrent `Send` and one concurrent `Receive` per `Connection` (atomic guards)
- `Receive` reads from channel, not socket — always goes through demuxer
- Client connections auto-reconnect; server (wrapped) connections do not
- Pool uses reference counting — physical connection closes when last logical connection releases

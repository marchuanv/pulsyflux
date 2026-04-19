---
inclusion: auto
---

# PulsyFlux Project Overview

PulsyFlux is a high-performance pub/sub message broker system written in Go with Node.js bindings via a native C++ addon.

## Repository Structure

- `tcp-conn/` — Low-level TCP connection abstraction (Go package `tcpconn`)
- `broker/` — Channel-based pub/sub message broker built on tcp-conn (Go package `broker`)
- `nodejs-api/` — Node.js native addon bindings (C++ addon + Go shared library + ES module wrapper)

## Module Info

- Go module: `pulsyflux` (go 1.25)
- Dependency: `github.com/google/uuid v1.6.0`
- Node.js package: `pulsyflux-broker` (published to npm)

## Architecture Layers

### tcp-conn (Transport Layer)

Provides multiplexed TCP connections with:
- Global connection pool — multiple logical connections share one physical TCP socket per address
- UUID-based message routing via a shared demuxer goroutine per physical connection
- Message framing: `[ID Length (1B)][ID (N B)][Total Length (4B)][Chunk Length (4B)][Data (M B)]`
- Chunked I/O (64KB chunks) for large messages
- Auto-reconnect on client side, no reconnect on server side (wrapped connections)
- Idle timeout (30s default) with automatic cleanup
- Thread-safe: atomic read/write guards, RWMutex on pool and routes

Key types:
- `Connection` — logical connection (client via `NewConnection`, server via `WrapConnection`)
- `demuxer` — single-reader goroutine per physical connection, routes by UUID
- `physicalPool` / `wrappedPool` — connection pooling with reference counting

### broker (Application Layer)

Lightweight pub/sub on top of tcp-conn:
- `Server` — accepts TCP connections, manages channels (map of UUID → channel)
- `Client` — connects to a channel, publishes/subscribes
- Control handshake: client sends `{client_id, channel_id}` on GlobalControlUUID, server acks
- Channel isolation: messages only broadcast within same channel
- Sender exclusion: publishers don't receive their own messages
- Subscribe returns buffered channel (capacity 100), non-blocking delivery (drops on full)

### nodejs-api (Bindings Layer)

- `broker_lib.go` — cgo exports (`ServerNew`, `ServerStart`, `ServerAddr`, `ServerStop`, `NewClient`, `Publish`, `Subscribe`, `FreePayload`, `Cleanup`)
- `addon.cc` — N-API C++ addon that loads `broker_lib.dll` and exposes `Server`/`Client` classes
- `registry.mjs` — ES module wrapper that re-exports native classes with a JS `Client` wrapper adding `onMessage()` support
- `types/index.d.ts` — TypeScript declarations
- Build: `zig-build` for cross-compilation, outputs to `.bin/release/`

## Testing

- Go broker tests: `cd broker && go test -v` (TestBasicPubSub, TestMultipleChannels)
- Go broker benchmarks: `cd broker && go test -bench=. -benchmem`
- Go tcp-conn tests: `cd tcp-conn && go test -v`
- Go tcp-conn benchmarks: `cd tcp-conn && go test -bench=. -benchmem`
- Node.js tests: Jasmine specs in `nodejs-api/spec/` (broker.spec.mjs, broker-benchmark.spec.mjs)

## Build

- Go shared library: `cd nodejs-api && go build -buildmode=c-shared -o .bin/release/broker_lib.dll broker_lib.go`
- C++ addon: `cd nodejs-api && node build.mjs` (uses zig-build)
- Build script: `nodejs-api/build.bat` / `nodejs-api/clean.bat`

## Performance Characteristics

- tcp-conn: ~39µs small message latency, ~8µs send/receive, supports 1MB+ messages
- broker: ~7µs publish, ~42µs round-trip pub/sub, ~24K msg/sec single core
- Node.js: ~13µs publish latency, 50K-76K publish ops/sec, ~20ms receive latency (AsyncWorker polling)

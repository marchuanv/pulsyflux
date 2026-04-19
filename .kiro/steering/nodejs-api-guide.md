---
inclusion: fileMatch
fileMatchPattern: "nodejs-api/**"
---

# Node.js API Guide

## Key Files

- `broker_lib.go` — cgo exports: `ServerNew`, `ServerStart`, `ServerAddr`, `ServerStop`, `NewClient`, `Publish`, `Subscribe`, `FreePayload`, `Cleanup`
- `addon.cc` — N-API C++ addon: loads `broker_lib.dll`, exposes `Server`/`Client` classes, `MessageWorker` for async receive
- `registry.mjs` — ES module wrapper: re-exports native `Server`, wraps native `Client` with JS `Client` class adding `onMessage()`
- `types/index.d.ts` — TypeScript declarations for `Server` and `Client`
- `build.mjs` — zig-build script for compiling C++ addon
- `spec/broker.spec.mjs` — Jasmine functional tests
- `spec/broker-benchmark.spec.mjs` — Jasmine performance benchmarks

## Build Pipeline

1. Go shared library: `go build -buildmode=c-shared -o .bin/release/broker_lib.dll broker_lib.go`
2. C++ addon: `node build.mjs` (zig-build, targets x86_64-windows, outputs `broker_addon.node`)
3. Published artifacts in `.bin/release/`: `broker_lib.dll`, `broker_addon.node`, `registry.mjs`

## Native Addon Architecture

```
Node.js → registry.mjs → broker_addon.node (C++ N-API) → broker_lib.dll (Go cgo)
```

- C++ addon loads DLL via `LoadLibraryA`, resolves function pointers via `GetProcAddress`
- Go side manages instances in global maps (`servers`, `clients`, `subs`) keyed by integer IDs
- Memory: Go allocates with `C.CBytes`, C++ copies to `Napi::Buffer`, caller frees with `FreePayload`

## MessageWorker (Async Receive)

- `MessageWorker` extends `Napi::AsyncWorker`
- Polls `Subscribe()` in background thread: 100 iterations × 1ms sleep = ~100ms polling window
- On message: calls JS callback, schedules next worker (continuous polling loop)
- Stored as `Napi::FunctionReference` on Client instance

## API Surface

- `Server(address)` — `.start()`, `.addr()`, `.stop()`
- `Client(address, channelID)` — `.publish(string|Buffer)`, `.subscribe()` (polling), `.onMessage(callback)` (event-driven), `.close()` (no-op)

## Testing

- Run specs: `npx jasmine` from `nodejs-api/`
- Tests require built artifacts in `.bin/release/`
- Use `addon.cleanup()` in `afterAll` to clean up Go runtime resources
- Use `setTimeout` delays for connection setup in async tests

## Platform

- Currently Windows-only (DLL loading, `LoadLibraryA`)
- Cross-platform would need `.so`/`.dylib` builds and platform-specific loading in `addon.cc`
- Package published as `pulsyflux-broker` on npm (os: win32, cpu: x64)

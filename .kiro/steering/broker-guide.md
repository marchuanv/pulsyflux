---
inclusion: auto
---

# Broker

A lightweight, channel-based pub/sub message broker built on [tcp](https://github.com/pulsyflux/tcp) multiplexing.

## Module

- Go module: `github.com/pulsyflux/broker` (go 1.25)
- Dependencies: `github.com/google/uuid v1.6.0`, `github.com/pulsyflux/tcp v0.1.1`

## Key Files

- `broker/server.go` — `Server` struct, accept loop, control message handling, channel management, broadcast
- `broker/client.go` — `Client` struct, `NewClient`, `Publish`, `Subscribe`, receive loop
- `broker/broker_test.go` — TestBasicPubSub, TestMultipleChannels
- `broker/broker_bench_test.go` — BenchmarkPublish, BenchmarkPubSub, BenchmarkBroadcast2/5/10, BenchmarkMultipleChannels

## Architecture

- `Server` — accepts TCP connections, manages channels (`map[uuid.UUID]*channel`)
- `Client` — connects to a channel, publishes/subscribes
- Control handshake: client sends `{client_id, channel_id}` on GlobalControlUUID, server acks with byte `0x01`
- Channel isolation: messages only broadcast within same channel
- Sender exclusion: publishers don't receive their own messages
- Subscribe returns buffered channel (capacity 100), non-blocking delivery (drops on full)
- 30s idle timeout inherited from [tcp](https://github.com/pulsyflux/tcp)

## Server Internals

- `channels`: `map[uuid.UUID]*channel` — each channel has its own `clients` map of `uuid.UUID → *tcpconn.Connection`
- Accept loop runs in goroutine, spawns `handleClient` per connection
- `handleClient` wraps connection with `GlobalControlUUID` via `tcpconn.WrapConnection`, loops receiving control messages
- On control message: creates/gets channel, wraps connection with clientID, spawns `handleChannel`
- `handleChannel` loops receiving on channel connection, calls `broadcast` for each message
- `broadcast` iterates channel clients under RLock, skips sender, sends to all others
- Shutdown via `done` channel — `Stop()` closes it, `acceptLoop` checks it each iteration

## Client Internals

- `NewClient(address, channelID)`: generates random clientID, creates control connection via `tcpconn.NewConnection(address, GlobalControlUUID)`, sends JSON join message, waits for ack, creates channel connection via `tcpconn.NewConnection(address, clientID)`, starts `receiveLoop`
- `receiveLoop`: reads from channel connection via `conn.Receive()`, fans out to all subscriber channels (non-blocking send with `select`/`default`)
- `Subscribe`: returns new buffered channel (cap 100), appended to `subs` slice under write lock
- `Publish`: sends directly on channel connection via `conn.Send(payload)`

## Control Handshake

1. Client creates connection with `GlobalControlUUID` (`00000000-0000-0000-0000-000000000000`)
2. Client sends JSON: `{"client_id": "<uuid>", "channel_id": "<uuid>"}`
3. Server registers client in channel, wraps connection with clientID
4. Server sends ack byte `0x01` on control connection
5. Client receives ack, creates channel connection with clientID

## Design Decisions

- One client per channel (create multiple clients for multiple channels)
- Raw bytes payload (no serialization — app's choice)
- Sender exclusion (publishers don't get their own messages)
- Non-blocking subscriber delivery (drops messages if subscriber channel full)

## Testing

- `go test ./broker/...`
- `go test -bench=. -benchmem ./broker/...`

## Current Limitations

- No message persistence or replay
- No delivery guarantees (fire-and-forget)
- No authentication/authorization
- No encryption (use TLS proxy)
- Single server (no clustering)
- Slow subscribers drop messages

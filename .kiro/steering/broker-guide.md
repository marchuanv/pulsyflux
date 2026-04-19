---
inclusion: fileMatch
fileMatchPattern: "broker/**"
---

# Broker Package Guide

## Key Files

- `server.go` — `Server` struct, accept loop, control message handling, channel management, broadcast
- `client.go` — `Client` struct, `NewClient`, `Publish`, `Subscribe`, receive loop
- `broker_test.go` — TestBasicPubSub, TestMultipleChannels
- `broker_bench_test.go` — BenchmarkPublish, BenchmarkPubSub, BenchmarkBroadcast2/5/10, BenchmarkMultipleChannels

## Server Internals

- `channels`: `map[uuid.UUID]*channel` — each channel has its own `clients` map of `uuid.UUID → *tcpconn.Connection`
- `clients`: `map[uuid.UUID]net.Conn` — tracks raw connections by client ID
- Accept loop runs in goroutine, spawns `handleClient` per connection
- `handleClient` creates a control connection (GlobalControlUUID), loops receiving control messages
- On control message: creates/gets channel, wraps connection with clientID, spawns `handleChannel`
- `handleChannel` loops receiving on channel connection, calls `broadcast` for each message
- `broadcast` iterates channel clients under RLock, skips sender, sends to all others

## Client Internals

- `NewClient`: generates random clientID, creates control connection, sends join message, waits for ack, creates channel connection, starts receiveLoop
- `receiveLoop`: reads from channel connection, fans out to all subscriber channels (non-blocking send)
- `Subscribe`: returns new buffered channel (cap 100), appended to `subs` slice
- `Publish`: sends directly on channel connection via `conn.Send`

## Control Handshake

1. Client creates connection with GlobalControlUUID (`00000000-...`)
2. Client sends JSON: `{"client_id": "<uuid>", "channel_id": "<uuid>"}`
3. Server registers client in channel, wraps connection with clientID
4. Server sends ack byte `0x01` on control connection
5. Client receives ack, creates channel connection with clientID

## Design Decisions

- One client per channel (create multiple clients for multiple channels)
- Raw bytes payload (no serialization — app's choice)
- Sender exclusion (publishers don't get their own messages)
- Non-blocking subscriber delivery (drops messages if subscriber channel full)
- 30s idle timeout inherited from tcp-conn

## Current Limitations

- No message persistence or replay
- No delivery guarantees (fire-and-forget)
- No authentication/authorization
- No encryption (use TLS proxy)
- Single server (no clustering)
- Slow subscribers drop messages

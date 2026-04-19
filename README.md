# Broker

A lightweight, channel-based pub/sub message broker built on [tcp](https://github.com/pulsyflux/tcp) multiplexing.

## Quick Start

```go
import (
    "github.com/google/uuid"
    "github.com/pulsyflux/broker/broker"
)

// Start server
server := broker.NewServer(":0")
server.Start()
defer server.Stop()

// Create clients on a channel
channelID := uuid.New()
client1, _ := broker.NewClient(server.Addr(), channelID)
client2, _ := broker.NewClient(server.Addr(), channelID)

// Subscribe and publish
ch := client2.Subscribe()
client1.Publish([]byte("hello"))
msg := <-ch // "hello" — sender does NOT receive own message
```

## Install

```bash
go get github.com/pulsyflux/broker
```

## Documentation

Detailed docs live in [`.kiro/steering/`](.kiro/steering/):

- [Broker Guide](.kiro/steering/broker-guide.md) — architecture, internals, design decisions, performance
- [Coding Standards](.kiro/steering/coding-standards.md) — Go conventions, thread safety, testing patterns
- [CI Workflow Guide](.kiro/steering/ci-workflow-guide.md) — CI rules, local testing with Act

## Testing

```bash
go test ./broker/...
go test -bench=. -benchmem ./broker/...
```

## Related Projects

- [tcp](https://github.com/pulsyflux/tcp) — TCP connection abstraction with multiplexing and connection pooling

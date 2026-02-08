# Socket Package - Best Practices Guide

## Table of Contents
1. [Architecture Overview](#architecture-overview)
2. [Connection Management](#connection-management)
3. [Error Handling](#error-handling)
4. [Performance Optimization](#performance-optimization)
5. [Concurrency Patterns](#concurrency-patterns)
6. [Testing Strategies](#testing-strategies)
7. [Production Deployment](#production-deployment)
8. [Common Pitfalls](#common-pitfalls)

## Architecture Overview

### Design Principles

The socket package follows these core principles:

1. **Broker Pattern**: Server acts as a message broker, not a direct peer
2. **Channel Isolation**: Each channel is independent with its own consumer/provider pairs
3. **Streaming First**: All payloads are streamed, no size limits
4. **Worker Pool**: RequestID hashing ensures ordered processing per request
5. **Async I/O**: Non-blocking writes with dedicated writer goroutines

### When to Use This Package

✅ **Good Use Cases:**
- Request-response RPC systems
- Microservice communication with load balancing
- Job queue systems with multiple workers
- Real-time data processing pipelines
- API gateway to backend services

❌ **Not Suitable For:**
- Pub/sub messaging (use NATS, Redis)
- Persistent message queues (use RabbitMQ, Kafka)
- Direct peer-to-peer communication
- Broadcasting to multiple consumers

## Connection Management

### Server Lifecycle

```go
// ✅ GOOD: Proper server lifecycle
server := socket.NewServer("9090")
if err := server.Start(); err != nil {
    log.Fatal(err)
}
defer server.Stop() // Always stop server

// Handle shutdown gracefully
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan
server.Stop()
```

```go
// ❌ BAD: No cleanup
server := socket.NewServer("9090")
server.Start()
// Server never stopped, goroutines leak
```

### Consumer Best Practices

```go
// ✅ GOOD: Reuse consumer for multiple requests
consumer, err := socket.NewConsumer("127.0.0.1:9090", channelID)
if err != nil {
    return err
}
defer consumer.Close()

for i := 0; i < 100; i++ {
    resp, err := consumer.Send(data, 5*time.Second)
    // Process response
}
```

```go
// ❌ BAD: Creating new consumer per request
for i := 0; i < 100; i++ {
    consumer, _ := socket.NewConsumer("127.0.0.1:9090", channelID)
    consumer.Send(data, 5*time.Second)
    consumer.Close() // Expensive connection overhead
}
```

### Provider Best Practices

```go
// ✅ GOOD: Dedicated goroutine for receiving
provider, err := socket.NewProvider("127.0.0.1:9090", channelID)
if err != nil {
    return err
}
defer provider.Close()

go func() {
    for {
        reqID, r, ok := provider.Receive()
        if !ok {
            return // Provider closed
        }
        
        // Process in separate goroutine for concurrency
        go handleRequest(provider, reqID, r)
    }
}()
```

```go
// ❌ BAD: Blocking receive in main goroutine
provider, _ := socket.NewProvider("127.0.0.1:9090", channelID)
reqID, r, _ := provider.Receive() // Blocks forever
// Rest of code never executes
```

### Channel ID Management

```go
// ✅ GOOD: Consistent channel IDs
const ServiceChannelID = "550e8400-e29b-41d4-a716-446655440000"
channelID := uuid.MustParse(ServiceChannelID)

// Or generate once and share
channelID := uuid.New()
// Store in config/database for reuse
```

```go
// ❌ BAD: Random channel IDs
channelID := uuid.New() // Different every time
// Consumer and provider won't connect
```

## Error Handling

### Timeout Handling

```go
// ✅ GOOD: Appropriate timeouts
resp, err := consumer.Send(data, 30*time.Second) // Long-running task
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Warn("Request timed out, retrying...")
        // Implement retry logic
    }
    return err
}
```

```go
// ❌ BAD: Too short timeout
resp, err := consumer.Send(data, 100*time.Millisecond)
// Likely to timeout on any real work
```

### Provider Error Responses

```go
// ✅ GOOD: Explicit error responses
reqID, r, ok := provider.Receive()
if !ok {
    return
}

data, err := processRequest(r)
if err != nil {
    provider.Respond(reqID, nil, err) // Send error to consumer
    return
}
provider.Respond(reqID, bytes.NewReader(data), nil)
```

```go
// ❌ BAD: Silent failures
reqID, r, _ := provider.Receive()
data, err := processRequest(r)
if err != nil {
    return // Consumer hangs waiting for response
}
provider.Respond(reqID, bytes.NewReader(data), nil)
```

### Connection Failures

```go
// ✅ GOOD: Retry with backoff
func connectWithRetry(addr string, channelID uuid.UUID) (*socket.Consumer, error) {
    backoff := time.Second
    for i := 0; i < 5; i++ {
        consumer, err := socket.NewConsumer(addr, channelID)
        if err == nil {
            return consumer, nil
        }
        log.Printf("Connection failed, retrying in %v...", backoff)
        time.Sleep(backoff)
        backoff *= 2
    }
    return nil, errors.New("max retries exceeded")
}
```

## Performance Optimization

### Payload Size Considerations

```go
// ✅ GOOD: Stream large payloads
file, _ := os.Open("large_file.dat")
defer file.Close()
resp, err := consumer.Send(file, 60*time.Second)
// Automatically chunked, no memory spike
```

```go
// ❌ BAD: Loading entire file into memory
data, _ := os.ReadFile("large_file.dat") // 1GB in memory
resp, err := consumer.Send(bytes.NewReader(data), 60*time.Second)
// High memory usage
```

### Connection Pooling

```go
// ✅ GOOD: Pool of consumers for high throughput
type ConsumerPool struct {
    consumers []*socket.Consumer
    mu        sync.Mutex
    idx       int
}

func (p *ConsumerPool) Get() *socket.Consumer {
    p.mu.Lock()
    defer p.mu.Unlock()
    c := p.consumers[p.idx]
    p.idx = (p.idx + 1) % len(p.consumers)
    return c
}

// Create pool
pool := &ConsumerPool{consumers: make([]*socket.Consumer, 10)}
for i := 0; i < 10; i++ {
    pool.consumers[i], _ = socket.NewConsumer(addr, channelID)
}
```

### Concurrent Request Processing

```go
// ✅ GOOD: Process requests concurrently
go func() {
    for {
        reqID, r, ok := provider.Receive()
        if !ok {
            return
        }
        
        // Each request in own goroutine
        go func(id uuid.UUID, reader io.Reader) {
            result := processRequest(reader)
            provider.Respond(id, result, nil)
        }(reqID, r)
    }
}()
```

```go
// ❌ BAD: Sequential processing
for {
    reqID, r, _ := provider.Receive()
    result := processRequest(r) // Blocks next request
    provider.Respond(reqID, result, nil)
}
```

## Concurrency Patterns

### Multiple Providers (Load Balancing)

```go
// ✅ GOOD: Multiple providers on same channel
channelID := uuid.New()

// Start 5 providers
for i := 0; i < 5; i++ {
    go func(id int) {
        provider, _ := socket.NewProvider(addr, channelID)
        defer provider.Close()
        
        for {
            reqID, r, ok := provider.Receive()
            if !ok {
                return
            }
            // Process and respond
            provider.Respond(reqID, processRequest(r), nil)
        }
    }(i)
}

// Consumers automatically load balanced across providers
```

### Multiple Channels (Service Isolation)

```go
// ✅ GOOD: Separate channels per service
authChannel := uuid.MustParse("...")
dataChannel := uuid.MustParse("...")

authProvider, _ := socket.NewProvider(addr, authChannel)
dataProvider, _ := socket.NewProvider(addr, dataChannel)

// Isolated traffic, no cross-contamination
```

### Request Cancellation

```go
// ✅ GOOD: Use context for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Send request in goroutine
respCh := make(chan io.Reader, 1)
errCh := make(chan error, 1)

go func() {
    resp, err := consumer.Send(data, 10*time.Second)
    if err != nil {
        errCh <- err
        return
    }
    respCh <- resp
}()

select {
case <-ctx.Done():
    return ctx.Err()
case err := <-errCh:
    return err
case resp := <-respCh:
    return processResponse(resp)
}
```

## Testing Strategies

### Unit Testing

```go
// ✅ GOOD: Test with embedded server
func TestConsumerProvider(t *testing.T) {
    server := socket.NewServer("0") // Random port
    server.Start()
    defer server.Stop()
    
    channelID := uuid.New()
    provider, _ := socket.NewProvider("127.0.0.1:"+server.Port(), channelID)
    defer provider.Close()
    
    // Test logic
}
```

### Integration Testing

```go
// ✅ GOOD: Test realistic scenarios
func TestHighLoad(t *testing.T) {
    server := socket.NewServer("9090")
    server.Start()
    defer server.Stop()
    
    // Start provider
    provider, _ := socket.NewProvider("127.0.0.1:9090", channelID)
    go handleRequests(provider)
    
    // Simulate 100 concurrent consumers
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            consumer, _ := socket.NewConsumer("127.0.0.1:9090", channelID)
            defer consumer.Close()
            
            for j := 0; j < 10; j++ {
                consumer.Send(data, 5*time.Second)
            }
        }()
    }
    wg.Wait()
}
```

### Benchmark Testing

```go
// ✅ GOOD: Measure performance
func BenchmarkThroughput(b *testing.B) {
    server := socket.NewServer("9090")
    server.Start()
    defer server.Stop()
    
    provider, _ := socket.NewProvider("127.0.0.1:9090", channelID)
    go echoProvider(provider)
    
    consumer, _ := socket.NewConsumer("127.0.0.1:9090", channelID)
    defer consumer.Close()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        consumer.Send(testData, 5*time.Second)
    }
}
```

## Production Deployment

### Monitoring

```go
// ✅ GOOD: Add metrics
type MetricsProvider struct {
    provider *socket.Provider
    requests prometheus.Counter
    latency  prometheus.Histogram
}

func (m *MetricsProvider) Handle() {
    for {
        start := time.Now()
        reqID, r, ok := m.provider.Receive()
        if !ok {
            return
        }
        
        m.requests.Inc()
        result := processRequest(r)
        m.provider.Respond(reqID, result, nil)
        m.latency.Observe(time.Since(start).Seconds())
    }
}
```

### Health Checks

```go
// ✅ GOOD: Implement health endpoint
func healthCheck(consumer *socket.Consumer) error {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    respCh := make(chan error, 1)
    go func() {
        _, err := consumer.Send(strings.NewReader("ping"), 2*time.Second)
        respCh <- err
    }()
    
    select {
    case <-ctx.Done():
        return errors.New("health check timeout")
    case err := <-respCh:
        return err
    }
}
```

### Graceful Shutdown

```go
// ✅ GOOD: Drain connections before shutdown
func gracefulShutdown(server *socket.Server, providers []*socket.Provider) {
    // Stop accepting new connections
    server.Stop()
    
    // Close all providers
    for _, p := range providers {
        p.Close()
    }
    
    // Wait for in-flight requests
    time.Sleep(5 * time.Second)
}
```

## Common Pitfalls

### ❌ Pitfall 1: Not Closing Connections

```go
// BAD
consumer, _ := socket.NewConsumer(addr, channelID)
consumer.Send(data, 5*time.Second)
// Forgot to close - connection leak
```

**Solution**: Always use `defer Close()`

### ❌ Pitfall 2: Blocking Provider.Receive()

```go
// BAD
provider, _ := socket.NewProvider(addr, channelID)
reqID, r, _ := provider.Receive() // Blocks main goroutine
```

**Solution**: Call `Receive()` in dedicated goroutine

### ❌ Pitfall 3: Ignoring Errors

```go
// BAD
resp, _ := consumer.Send(data, 5*time.Second)
// Ignored error, resp might be nil
```

**Solution**: Always check errors

### ❌ Pitfall 4: Wrong Channel ID

```go
// BAD
consumer, _ := socket.NewConsumer(addr, uuid.New())
provider, _ := socket.NewProvider(addr, uuid.New())
// Different channels, won't communicate
```

**Solution**: Share channel ID between consumer and provider

### ❌ Pitfall 5: Timeout Too Short

```go
// BAD
resp, _ := consumer.Send(largeData, 100*time.Millisecond)
// Likely to timeout
```

**Solution**: Set timeout based on expected processing time

## Summary

**Key Takeaways:**
1. Always close connections with `defer`
2. Use dedicated goroutines for `Provider.Receive()`
3. Handle errors explicitly
4. Set appropriate timeouts
5. Reuse connections when possible
6. Process requests concurrently in providers
7. Monitor and measure in production
8. Test under realistic load

**Performance Tips:**
- Pool consumers for high throughput
- Stream large payloads
- Use multiple providers for load balancing
- Pre-allocate buffers where possible
- Monitor memory and goroutine counts

**Production Checklist:**
- [ ] Health checks implemented
- [ ] Metrics and logging added
- [ ] Graceful shutdown handling
- [ ] Connection retry logic
- [ ] Timeout tuning completed
- [ ] Load testing performed
- [ ] Error handling verified
- [ ] Resource cleanup confirmed

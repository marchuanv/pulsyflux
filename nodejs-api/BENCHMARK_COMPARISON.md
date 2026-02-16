# Benchmark Comparison: Go vs Node.js

## Performance Comparison

### Single Request/Response (Round-trip)

| Metric | Go (Native) | Node.js (FFI) | Overhead |
|--------|-------------|---------------|----------|
| Latency | 45µs | 6,573µs (6.6ms) | ~145x |
| Throughput | ~22K ops/sec | 152 ops/sec | ~145x slower |

### Publish Only

| Metric | Go (Native) | Node.js (FFI) | Overhead |
|--------|-------------|---------------|----------|
| Latency | 7µs | ~1,250µs (1.25ms) | ~178x |
| Throughput | ~142K ops/sec | 801 ops/sec | ~177x slower |

### Broadcast (5 clients)

| Metric | Go (Native) | Node.js (FFI) | Overhead |
|--------|-------------|---------------|----------|
| Latency | 57µs | N/A | N/A |
| Throughput | ~17.5K ops/sec | 40 deliveries/sec | ~437x slower |

### Large Payload (1MB)

| Metric | Go (Native) | Node.js (FFI) | Notes |
|--------|-------------|---------------|-------|
| Latency | N/A | 34ms | Includes FFI marshaling |
| Bandwidth | N/A | 58.8 MB/s | Good for large payloads |

### Medium Payload (10KB)

| Metric | Go (Native) | Node.js (FFI) | Overhead |
|--------|-------------|---------------|----------|
| Latency | N/A | 7.65ms | FFI + marshaling |
| Throughput | N/A | 131 ops/sec | Reasonable |

## Analysis

### FFI Overhead

The Node.js bindings add significant overhead:
- **Small messages**: ~145-178x slower than native Go
- **Overhead breakdown**:
  - FFI call: ~1-2µs per call
  - Buffer marshaling: ~1-5µs depending on size
  - JavaScript event loop: ~1-2ms
  - Total: ~6-7ms for round-trip

### When to Use Each

**Use Go Broker (Native):**
- High-frequency trading systems
- Real-time gaming
- Low-latency microservices
- Performance-critical applications
- Need <100µs latency

**Use Node.js Bindings:**
- Node.js/JavaScript applications
- Web applications
- Moderate throughput requirements (<1K ops/sec)
- Integration with existing Node.js ecosystem
- Acceptable latency >5ms

### Strengths of Node.js Bindings

Despite the overhead:
- ✅ Still achieves **sub-10ms latency** for most operations
- ✅ **800+ ops/sec** throughput is sufficient for many applications
- ✅ **58 MB/s bandwidth** for large payloads
- ✅ No Python required for installation
- ✅ Easy integration with Node.js applications
- ✅ Async/await support with iterators

### Bottlenecks

1. **FFI overhead**: ~1-2µs per call (unavoidable)
2. **Buffer marshaling**: ~1-5µs (depends on payload size)
3. **JavaScript event loop**: ~1-2ms (polling interval)
4. **Receive polling**: 10ms intervals in async iterator

### Optimization Opportunities

To improve Node.js performance:
1. Reduce polling interval (currently 10ms)
2. Batch operations where possible
3. Use larger payloads to amortize FFI overhead
4. Consider native Node.js addon instead of FFI

## Conclusion

**The Node.js bindings have UNACCEPTABLE performance for a message broker.**

Achieving only:
- **6-7ms round-trip latency** (vs 45µs native) - **145x slower**
- **800 ops/sec throughput** (vs 22K native) - **27x lower throughput**
- **40 deliveries/sec broadcast** (vs 17.5K native) - **437x slower**

### Why This Matters

Message brokers are **performance-critical infrastructure**:
- Used for inter-service communication
- Often on the critical path of requests
- Need to handle high message volumes
- Latency directly impacts user experience

### The Problem

**FFI overhead is too high:**
- Each message requires multiple FFI calls
- Buffer marshaling adds milliseconds
- JavaScript event loop adds latency
- Polling-based receive is inefficient

### Recommendation

**DO NOT use the Node.js bindings for production message broker workloads.**

Instead:
1. **Use the Go broker directly** - Deploy as a service
2. **Use native Node.js clients** - Connect via TCP/WebSocket
3. **Consider alternatives** - Redis, NATS, RabbitMQ have optimized Node.js clients
4. **Rewrite as native addon** - If Node.js integration is required

### When Node.js Bindings Are Acceptable

- **Testing/development only**
- **Very low throughput** (<100 msgs/sec)
- **Non-critical paths** (logging, metrics)
- **Prototyping** before production deployment

## Rating

**Node.js Bindings: D (Poor for Production)**
- ❌ 145x slower than native - UNACCEPTABLE
- ❌ 800 ops/sec - Too low for message broker
- ❌ FFI overhead dominates performance
- ✅ Works for testing/development
- ⚠️ **NOT RECOMMENDED for production use**

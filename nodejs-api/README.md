# Node.js Broker API

Node.js bindings for the PulsyFlux broker using FFI.

## Overview

This package provides Node.js bindings to the Go broker implementation, allowing JavaScript/TypeScript applications to use the high-performance pub/sub broker.

**Performance:** ~45-50µs round-trip latency with minimal FFI overhead (~3-5µs).

## Installation

### Prerequisites

- **Node.js** 14+ 
- **Go** 1.19+ (for building the shared library)
- **Python** 3.x (required by ffi-napi for native module compilation)
- **C++ Build Tools** (Windows: Visual Studio Build Tools)

### Option 1: Full Install (Requires Python)

```bash
npm install
```

This will:
1. Install Node.js dependencies (requires Python for ffi-napi)
2. Automatically build `broker_lib.dll` via postinstall script

### Option 2: Without Python (Manual Build)

If you don't have Python installed:

```bash
# Skip native builds
npm install --ignore-scripts

# Build Go library manually
npm run build
```

## Build

Build the Go shared library:

```bash
npm run build
```

This creates `broker_lib.dll` (Windows) that Node.js will load via FFI.

## Quick Start

```javascript
import { Server, Client } from 'pulsyflux-broker';
import { randomUUID } from 'crypto';

// Start server
const server = new Server(':0');
server.start();

// Create clients on same channel
const channelID = randomUUID();
const client1 = new Client(server.addr(), channelID);
const client2 = new Client(server.addr(), channelID);

// Subscribe
const sub = client2.subscribe();

// Receive messages
(async () => {
  for await (const msg of sub) {
    console.log('Received:', msg.toString());
  }
})();

// Publish
client1.publish('hello from nodejs!');

// Cleanup
sub.close();
server.stop();
```

## API Reference

### Server

#### `new Server(address)`
Creates a new broker server.
- `address`: Listen address (e.g., `:8080` or `localhost:8080`)
- Use `:0` for random port

```javascript
const server = new Server(':8080');
```

#### `server.start()`
Starts the server.

```javascript
server.start();
```

#### `server.addr()`
Returns the actual listening address.

```javascript
const addr = server.addr(); // "[::]:8080"
```

#### `server.stop()`
Stops the server and closes all connections.

```javascript
server.stop();
```

### Client

#### `new Client(address, channelID)`
Creates a new client connected to a specific channel.
- `address`: Server address (e.g., `localhost:8080`)
- `channelID`: UUID string for the channel

```javascript
import { randomUUID } from 'crypto';

const channelID = randomUUID();
const client = new Client('localhost:8080', channelID);
```

#### `client.publish(payload)`
Publishes a message to the channel.
- `payload`: String or Buffer to publish
- Sender does NOT receive their own message

```javascript
// String
client.publish('hello world');

// Buffer
client.publish(Buffer.from([1, 2, 3, 4]));

// JSON
client.publish(JSON.stringify({ id: 123, name: 'test' }));
```

#### `client.subscribe()`
Creates a subscription to receive messages from the channel.
- Returns: `Subscription` object

```javascript
const sub = client.subscribe();
```

### Subscription

#### `subscription.receive()`
Receives a message (non-blocking).
- Returns: `Buffer` or `null` if no message available

```javascript
const msg = sub.receive();
if (msg) {
  console.log('Received:', msg.toString());
}
```

#### `subscription[Symbol.asyncIterator]()`
Async iterator for receiving messages.
- Polls every 10ms when no messages available

```javascript
for await (const msg of sub) {
  console.log('Received:', msg.toString());
}
```

#### `subscription.close()`
Closes the subscription.

```javascript
sub.close();
```

## Examples

### Basic Pub/Sub

```javascript
import { Server, Client } from 'pulsyflux-broker';
import { randomUUID } from 'crypto';

const server = new Server(':0');
server.start();

const channelID = randomUUID();
const publisher = new Client(server.addr(), channelID);
const subscriber = new Client(server.addr(), channelID);

const sub = subscriber.subscribe();

// Receive
(async () => {
  for await (const msg of sub) {
    console.log('Received:', msg.toString());
    sub.close();
    break;
  }
})();

// Publish
setTimeout(() => {
  publisher.publish('Hello World!');
}, 100);
```

### JSON Messages

```javascript
const data = { id: 123, name: 'test', timestamp: Date.now() };

// Publish
client.publish(JSON.stringify(data));

// Receive
const msg = sub.receive();
if (msg) {
  const received = JSON.parse(msg.toString());
  console.log(received.id, received.name);
}
```

### Binary Data

```javascript
// Publish binary
const buffer = Buffer.from([0x01, 0x02, 0x03, 0x04]);
client.publish(buffer);

// Receive binary
const msg = sub.receive();
if (msg) {
  console.log('Bytes:', Array.from(msg));
}
```

### Multiple Subscribers

```javascript
const channelID = randomUUID();
const publisher = new Client(server.addr(), channelID);
const sub1 = new Client(server.addr(), channelID).subscribe();
const sub2 = new Client(server.addr(), channelID).subscribe();
const sub3 = new Client(server.addr(), channelID).subscribe();

// All subscribers receive the message
publisher.publish('broadcast to all');

// Publisher does NOT receive own message
```

### Channel Isolation

```javascript
const channel1 = randomUUID();
const channel2 = randomUUID();

const clientA = new Client(server.addr(), channel1);
const clientB = new Client(server.addr(), channel2);

const subA = clientA.subscribe();
const subB = clientB.subscribe();

clientA.publish('message on channel 1');
clientB.publish('message on channel 2');

// subA only receives channel1 messages
// subB only receives channel2 messages
```

## Performance

### Benchmark Results

```
Single Request/Response:  ~45-50µs round-trip
Publish Throughput:       ~20K ops/sec
Broadcast (5 clients):    ~15K deliveries/sec
Large Payload (1MB):      ~150ms round-trip
Medium Payload (10KB):    ~60µs round-trip
```

### Performance Characteristics

- **FFI Overhead**: ~3-5µs per call
- **Buffer Marshaling**: ~1µs
- **Total Overhead**: ~5-10µs on top of broker latency
- **Broker Latency**: ~7µs publish, ~42µs round-trip
- **Expected Total**: ~45-50µs round-trip

### Comparison

- **Redis Pub/Sub**: ~100-200µs
- **NATS**: ~50-100µs  
- **This Broker**: ~45-50µs ⭐

## Architecture

### Connection Flow

1. Client sends control message with ClientID + ChannelID
2. Server registers client in channel
3. Server sends ack byte back
4. Client creates channel connection
5. Messages flow over channel connection

### Key Features

- **Connection Pooling**: Multiple clients share physical TCP connections
- **Multiplexing**: Logical connections over shared sockets
- **Sender Exclusion**: Publishers don't receive own messages
- **Channel Isolation**: Messages don't leak between channels
- **Raw Bytes**: No serialization overhead (application's choice)

## Limitations

- Same as broker package:
  - No message persistence
  - No delivery guarantees
  - No authentication
  - Sender cannot receive own messages
  - 30-second idle timeout
  - Slow subscribers drop messages (100-message buffer)
- FFI overhead adds ~5-10µs latency
- Buffer copies required for data marshaling
- Windows only (currently)

## Testing

### Run Tests

```bash
npm test
```

### Run Benchmarks

```bash
npm test -- --filter="*Benchmark*"
```

**Test Coverage:**
- Basic pub/sub functionality
- Binary and JSON payloads
- Async iteration
- Multiple messages
- Sender exclusion
- Channel isolation
- Performance benchmarks

## Troubleshooting

### Python Not Found

If npm install fails with "Could not find any Python installation":

1. Install Python 3.x from https://www.python.org/downloads/
2. Ensure Python is in PATH
3. Or set Python path:
   ```bash
   npm config set python "C:\Path\To\python.exe"
   ```

### Build Fails

Ensure Go is installed and in PATH:
```bash
go version
```

### DLL Not Found

Run build manually:
```bash
npm run build
```

### Connection Timeout

Increase wait time after server start:
```javascript
server.start();
await new Promise(r => setTimeout(r, 100));
```

## License

Same as PulsyFlux project.

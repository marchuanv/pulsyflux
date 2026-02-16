# Node.js Broker API

Node.js bindings for the PulsyFlux broker using FFI.

## Overview

This package provides Node.js bindings to the Go broker implementation, allowing JavaScript/TypeScript applications to use the high-performance pub/sub broker.

## Installation

```bash
npm install
```

## Build

Build the Go shared library:

```bash
build.bat
```

This creates `broker_lib.dll` (Windows) that Node.js will load via FFI.

## API

### Server

```javascript
import { Server } from './broker.mjs';

const server = new Server(':8080');
server.start();
const addr = server.addr(); // Get actual address
server.stop();
```

### Client

```javascript
import { Client } from './broker.mjs';
import { randomUUID } from 'crypto';

const channelID = randomUUID();
const client = new Client('localhost:8080', channelID);

// Publish
client.publish('hello world');
client.publish(Buffer.from([1, 2, 3, 4]));

// Subscribe
const sub = client.subscribe();

// Receive (blocking)
const msg = sub.receive(); // Returns Buffer or null

// Receive (async iterator)
for await (const msg of sub) {
  console.log('Received:', msg.toString());
}

// Cleanup
sub.close();
```

## Example

```javascript
import { Server, Client } from './broker.mjs';
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
- `address`: Listen address (e.g., `:8080` or `localhost:8080`)
- Use `:0` for random port

#### `server.start()`
Starts the server.

#### `server.addr()`
Returns the actual listening address.

#### `server.stop()`
Stops the server.

### Client

#### `new Client(address, channelID)`
- `address`: Server address
- `channelID`: UUID string for the channel

#### `client.publish(payload)`
- `payload`: String or Buffer to publish

#### `client.subscribe()`
Returns a `Subscription` object.

### Subscription

#### `subscription.receive()`
- Returns: Buffer or null (if no message available)
- Non-blocking

#### `subscription[Symbol.asyncIterator]()`
- Async iterator for receiving messages
- Polls every 10ms when no messages available

#### `subscription.close()`
Closes the subscription.

## Performance

The Node.js bindings have minimal overhead:
- FFI call overhead: ~1-2µs
- Buffer marshaling: ~1µs
- Total overhead: ~3-5µs on top of broker latency

Expected performance:
- Publish: ~10-12µs
- Round-trip: ~45-50µs

## Limitations

- Same as broker package (see broker README)
- FFI overhead adds ~3-5µs latency
- Buffer copies required for data marshaling
- No automatic reconnection (inherit from broker)

## Testing

```bash
npm test
```

## License

Same as PulsyFlux project.

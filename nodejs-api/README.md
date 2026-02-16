# MessageBus Node.js Bindings

Node.js bindings for the PulsyFlux MessageBus package using CGO and FFI.

## Architecture

```
Node.js Application
    ↓
FFI Bindings (messagebus.mjs)
    ↓
CGO Library (messagebus_lib.dll)
    ↓
Go MessageBus Package
    ↓
Go Socket Package
    ↓
TCP
```

## Build

```bash
# Build the shared library
build-messagebus.bat

# Install Node.js dependencies
npm install
```

## API

### Server

```javascript
import { Server } from './messagebus.mjs';

const server = new Server('9090');
server.start();
// ... use server
server.stop();
```

### Bus

```javascript
import { Bus } from './messagebus.mjs';

const bus = new Bus('127.0.0.1:9090', channelID);

// Publish
bus.publish('topic', 'payload', { key: 'value' }, 5000);

// Subscribe
const sub = bus.subscribe('topic');

// Unsubscribe
bus.unsubscribe('topic');

// Close
bus.close();
```

### Subscription

```javascript
// Poll for messages
const msg = sub.receive();
if (msg) {
  console.log(msg.topic, msg.payload);
}

// Async iterator
for await (const msg of sub) {
  console.log(msg.topic, msg.payload);
}

// Close
sub.close();
```

## Usage Examples

### Basic Pub/Sub

```javascript
import { Server, Bus } from './messagebus.mjs';
import { v4 as uuidv4 } from 'uuid';

// Start server
const server = new Server('9090');
server.start();

// Create bus
const channelID = uuidv4();
const bus = new Bus('127.0.0.1:9090', channelID);

// Subscribe
const sub = bus.subscribe('events');

// Receive messages
(async () => {
  for await (const msg of sub) {
    console.log('Received:', msg.payload.toString());
  }
})();

// Publish
bus.publish('events', 'Hello World!');
```

### With Headers

```javascript
bus.publish('orders.created', JSON.stringify({ id: 123 }), {
  user_id: '456',
  source: 'api',
  priority: 'high'
});

for await (const msg of sub) {
  console.log('User:', msg.headers.user_id);
  console.log('Data:', msg.payload.toString());
}
```

### Multiple Subscribers

```javascript
const sub1 = bus.subscribe('events');
const sub2 = bus.subscribe('events');

// Both receive the same messages
bus.publish('events', 'broadcast');
```

### Polling Pattern

```javascript
const sub = bus.subscribe('jobs');

setInterval(() => {
  const msg = sub.receive();
  if (msg) {
    processJob(msg.payload);
  }
}, 100);
```

### Event-Driven Pattern

```javascript
const sub = bus.subscribe('notifications');

(async () => {
  for await (const msg of sub) {
    await sendNotification(msg.payload);
  }
})();
```

## Message Structure

```javascript
{
  id: 'uuid',              // Message ID
  topic: 'string',         // Topic name
  payload: Buffer,         // Message data
  headers: Object,         // Metadata
  timestamp: Date          // Creation time
}
```

## Performance

- **Latency**: ~123µs (inherited from Go layer)
- **Throughput**: ~8K messages/sec
- **Overhead**: FFI + JSON serialization

## Comparison with socket-exp

| Feature | socket-exp | messagebus-nodejs |
|---------|-----------|-------------------|
| Layer | Socket (low-level) | MessageBus (high-level) |
| API | Consumer/Provider | Bus/Pub/Sub |
| Complexity | Manual routing | Automatic routing |
| Use Case | RPC | Event messaging |

## Migration from socket-exp

**Before (socket-exp):**
```javascript
const consumer = new Consumer(addr, channelID);
const response = consumer.send('request', 5000);
```

**After (messagebus-nodejs):**
```javascript
const bus = new Bus(addr, channelID);
bus.publish('topic', 'message');
const sub = bus.subscribe('topic');
const msg = sub.receive();
```

## Testing

```bash
node example-messagebus.mjs
```

## Troubleshooting

### Build Fails
- Ensure GCC is installed (MSYS2 mingw64)
- Check `C:\msys64\mingw64\bin` is in PATH
- Disable Windows Defender real-time protection temporarily

### FFI Errors
- Run `npm install --ignore-scripts` if native build fails
- Ensure DLL is in `build/` directory

### No Messages Received
- Check server is running
- Verify channel IDs match
- Check topic names are exact

## Files

- `messagebus_lib.go` - CGO exports
- `messagebus.mjs` - Node.js FFI bindings
- `example-messagebus.mjs` - Usage example
- `build-messagebus.bat` - Build script

## License

MIT

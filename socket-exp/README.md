# Socket Export for Node.js

Node.js bindings for the PulsyFlux socket package using CGO and FFI.

## Build

```bash
# Build the shared library
build.bat

# Files will be in build/ directory:
# - build/socket_lib.dll
# - socket_lib.h

# Install Node.js dependencies
npm install
```

## Usage

### Consumer

```javascript
const { Consumer } = require('./socket');

const consumer = new Consumer('127.0.0.1:9090', 'channel-uuid');
const response = consumer.send('Hello', 5000);
console.log(response);
consumer.close();
```

### Provider

```javascript
const { Provider } = require('./socket');

const provider = new Provider('127.0.0.1:9090', 'channel-uuid');

setInterval(() => {
  const req = provider.receive();
  if (req) {
    provider.respond(req.requestID, `Processed: ${req.data}`);
  }
}, 100);
```

## API

### Consumer

- `new Consumer(address, channelID)` - Create consumer
- `send(data, timeoutMs)` - Send request, returns response
- `close()` - Close connection

### Provider

- `new Provider(address, channelID)` - Create provider
- `receive()` - Receive request, returns `{requestID, data}` or `null`
- `respond(requestID, data, error)` - Send response
- `close()` - Close connection

## Files

- `socket_lib.go` - CGO exports
- `socket.js` - Node.js FFI bindings
- `example.js` - Usage examples
- `build.bat` - Build script
- `build/socket_lib.dll` - Compiled shared library
- `socket_lib.h` - C header file

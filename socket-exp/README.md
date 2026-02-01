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

## Implementation Notes

### CGO Bridge (`socket_lib.go`)

The CGO bridge provides C-compatible exports for the Go socket package:

- **Consumer functions**: `ConsumerNew`, `ConsumerSend`, `ConsumerClose`
- **Provider functions**: `ProviderNew`, `ProviderReceive`, `ProviderRespond`, `ProviderClose`
- **Server functions**: `ServerNew`, `ServerStop`
- **Memory management**: `FreeString` for cleanup

### Key Implementation Details

1. **Non-blocking Provider.Receive()**: The Go socket package's `Receive()` method is non-blocking and returns immediately. The CGO wrapper calls it directly without timeouts or goroutines.

2. **Memory Management**: Uses global maps to track consumer, provider, and server instances by ID. Proper cleanup on close operations.

3. **Build Process**: Must be built from parent directory (`e:\github\pulsyflux`) to access the socket package module. The `build.bat` handles this automatically.

4. **Module Structure**: 
   - Parent `go.mod` at `e:\github\pulsyflux` contains all module definitions
   - Socket-exp builds as part of the pulsyflux module
   - Uses `replace` directives for local package references

### FFI Bindings (`socket.js`)

Node.js FFI bindings using `ffi-napi` and `ref-napi`:

- **Consumer class**: Wraps CGO consumer functions
- **Provider class**: Extends EventEmitter, wraps CGO provider functions  
- **Server class**: Wraps CGO server functions
- **Error handling**: Throws errors for failed operations

### Testing

- `simple-test.js` - Basic latency test with echo server
- `simple.test.js` - Jest unit tests for object creation
- Provider polling pattern: Use `setInterval` with 10ms intervals for responsive request handling

### Build Requirements

- Go 1.21+
- CGO enabled
- MinGW64 (for Windows DLL compilation)
- Node.js with `ffi-napi` and `ref-napi` packages

### Troubleshooting

1. **Build failures**: Ensure building from parent directory with proper module context
2. **Import errors**: Verify socket package import path is `pulsyflux/socket`
3. **Runtime errors**: Check that server is started before creating consumers/providers
4. **Memory leaks**: Always call `close()` methods for proper cleanup

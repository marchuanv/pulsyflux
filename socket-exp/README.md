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

## Current Development Status

### Issue: Test Timeouts in Jasmine Test Suite

**Context**: The Node.js wrapper uses worker threads to bridge the synchronous FFI bindings with Node.js async patterns. Tests are run using Jasmine with the following test structure:

- `spec/socket.spec.mjs` - Main test suite with 9 test cases
- Tests cover: Consumer creation, Provider creation, request/response cycles, timeouts, multiple consumers, large payloads
- Server is started once in `beforeAll()` and reused across tests
- Each test uses a unique channel ID to avoid interference

**Problem**: Tests were failing with timeout errors where:
1. Providers successfully receive requests (confirmed in logs)
2. Providers successfully send responses (confirmed in logs)
3. Consumers timeout waiting for responses
4. Tests fail with "Consumer send timeout" and Jasmine timeout errors

**Root Cause Identified**: The provider polling mechanism in `wrapper.mjs` was using `setImmediate()` for polling, which created an infinite tight loop that blocked the Node.js event loop, preventing proper message passing between worker threads and the main thread.

**Fix Applied**: Changed polling from `setImmediate()` to `setTimeout(poll, 1)` with 1ms intervals to allow the event loop to process other operations while maintaining responsive polling.

**Current Status**: 
- Major improvement: Most tests now pass
- Socket communication is working (logs show successful request/response cycles)
- 2 tests still failing, but core timeout issue is resolved
- Need to investigate remaining edge cases

**Test Output Pattern**:
```
Provider X received request: reqID=..., payloadSize=...
Provider X responding with data: payloadSize=...
Provider X response sent successfully for reqID=...
Consumer X send successful: responseSize=...
```

### Architecture Overview

## Files

- `socket_lib.go` - CGO exports
- `socket.mjs` - Node.js FFI bindings (direct, synchronous)
- `wrapper.mjs` - Worker thread wrapper for async API
- `spec/socket.spec.mjs` - Jasmine test suite
- `example.js` - Usage examples
- `build.bat` - Build script
- `build/socket_lib.dll` - Compiled shared library
- `socket_lib.h` - C header file

### File Relationships

1. **Go Layer**: `socket_lib.go` exports Go socket package functions via CGO
2. **FFI Layer**: `socket.mjs` provides direct Node.js bindings to the DLL
3. **Wrapper Layer**: `wrapper.mjs` adds async/await support using worker threads
4. **Test Layer**: `spec/socket.spec.mjs` tests the wrapper API

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

### Worker Thread Wrapper (`wrapper.mjs`)

To handle the synchronous nature of the FFI bindings in an async Node.js environment, a worker thread wrapper is implemented:

- **Consumer wrapper**: Uses worker threads to make synchronous `send()` calls non-blocking
- **Provider wrapper**: Implements async polling for incoming requests using worker threads
- **Server wrapper**: Wraps server start/stop operations in worker threads

#### Current Issue: Test Timeouts

**Problem**: Tests are failing with timeout errors where providers receive requests but consumers don't get responses.

**Root Cause**: The provider polling mechanism in the worker thread wrapper was too aggressive, using `setImmediate()` which blocked the event loop and prevented proper request/response handling.

**Symptoms**:
- Providers successfully receive requests (logged)
- Providers successfully send responses (logged) 
- Consumers timeout waiting for responses
- Tests fail with "Consumer send timeout" errors

**Fix Applied**: Changed from aggressive `setImmediate()` polling to `setTimeout(poll, 1)` with 1ms intervals to allow the event loop to process other operations.

**Test Status**: Most tests now pass, but 2 tests still failing. The socket communication is working (logs show successful request/response cycles), indicating the core issue is resolved but some edge cases remain.

**Test Logs Show**:
```
Provider X received request: reqID=..., payloadSize=...
Provider X responding with data: payloadSize=...
Provider X response sent successfully for reqID=...
Consumer X send successful: responseSize=...
```

#### Worker Thread Architecture

1. **Main Thread**: Creates Consumer/Provider/Server wrapper classes
2. **Worker Thread**: Instantiates actual FFI objects and handles synchronous operations
3. **Message Passing**: Coordinates between main thread async API and worker thread sync operations
4. **Polling**: Provider wrapper polls for requests every 1ms and forwards to main thread
5. **Queue Management**: Provider maintains request queue and pending promise resolvers

### Troubleshooting

1. **Build failures**: Ensure building from parent directory with proper module context
2. **Import errors**: Verify socket package import path is `pulsyflux/socket`
3. **Runtime errors**: Check that server is started before creating consumers/providers
4. **Memory leaks**: Always call `close()` methods for proper cleanup
5. **Test timeouts**: If tests timeout, check that provider polling is not blocking the event loop
6. **Worker thread issues**: Ensure proper cleanup of worker threads on close operations

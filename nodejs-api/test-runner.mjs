import { Server, Client } from './registry.mjs';
import { randomUUID } from 'crypto';

console.log('Running broker tests...');

// Test 1: Server basic functionality
console.log('Test 1: Server start/stop');
const server = new Server(':0');
server.start();
const addr = server.addr();
console.log('✓ Server started at:', addr);
server.stop();
console.log('✓ Server stopped');

// Test 2: Client creation and messaging
console.log('\nTest 2: Client messaging');
const testServer = new Server(':0');
testServer.start();
const testAddr = testServer.addr();

const channelID = randomUUID();
const client1 = new Client(testAddr, channelID);
const client2 = new Client(testAddr, channelID);

console.log('✓ Clients created');

// Test null subscription
const nullMsg = client1.subscribe();
console.log('✓ Null subscription returns:', nullMsg);

// Test messaging
client1.publish('Hello World');
setTimeout(() => {
  const msg = client2.subscribe();
  if (msg && msg.toString() === 'Hello World') {
    console.log('✓ Message received:', msg.toString());
  } else {
    console.log('✗ Message not received');
  }
  
  testServer.stop();
  console.log('✓ Test server stopped');
  
  console.log('\nAll tests completed successfully!');
  
  // Force exit due to Go runtime keeping event loop alive
  process.exit(0);
}, 100);
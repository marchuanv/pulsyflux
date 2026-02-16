import { Server, Client } from './broker.mjs';
import { randomUUID } from 'crypto';

// Start server
const server = new Server(':0');
server.start();
const addr = server.addr();
console.log('Server started at:', addr);

// Create channel
const channelID = randomUUID();
console.log('Channel ID:', channelID);

// Create clients
const client1 = new Client(addr, channelID);
const client2 = new Client(addr, channelID);

// Subscribe client2
const sub = client2.subscribe();

// Receive messages
(async () => {
  for await (const msg of sub) {
    console.log('Client2 received:', msg.toString());
  }
})();

// Wait a bit for subscription to be ready
await new Promise(resolve => setTimeout(resolve, 100));

// Publish from client1
console.log('Client1 publishing...');
client1.publish('hello from nodejs!');

// Wait for message
await new Promise(resolve => setTimeout(resolve, 500));

// Cleanup
sub.close();
server.stop();
console.log('Done');

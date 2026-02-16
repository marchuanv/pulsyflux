import { Server, Bus } from './messagebus.mjs';
import { v4 as uuidv4 } from 'uuid';

// Start server
const server = new Server('9090');
server.start();
console.log('Server started on port 9090');

// Create bus
const channelID = uuidv4();
const bus = new Bus('127.0.0.1:9090', channelID);
console.log('Bus created with channel:', channelID);

// Subscribe to topic
const subscription = bus.subscribe('test.topic');
console.log('Subscribed to test.topic');

// Receive messages in background
(async () => {
  for await (const msg of subscription) {
    console.log('Received message:', {
      id: msg.id,
      topic: msg.topic,
      payload: msg.payload.toString(),
      headers: msg.headers,
      timestamp: msg.timestamp
    });
  }
})();

// Publish messages
setTimeout(() => {
  console.log('Publishing message...');
  bus.publish('test.topic', 'Hello from Node.js!', {
    source: 'example',
    priority: 'high'
  });
}, 1000);

// Cleanup after 5 seconds
setTimeout(() => {
  console.log('Cleaning up...');
  subscription.close();
  bus.close();
  server.stop();
  process.exit(0);
}, 5000);

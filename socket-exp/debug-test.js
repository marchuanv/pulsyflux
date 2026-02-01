const { Provider, Consumer, Server } = require('./socket');
const { v4: uuidv4 } = require('uuid');

console.log('Creating server...');
const server = new Server('9102');

const channelID = uuidv4();
console.log('Channel ID:', channelID);

console.log('Creating provider...');
const provider = new Provider('127.0.0.1:9102', channelID);

console.log('Creating consumer...');
const consumer = new Consumer('127.0.0.1:9102', channelID);

console.log('Setting up consumer timeout...');
// Send a message from consumer in background
setTimeout(() => {
  console.log('Consumer timeout fired - sending message...');
  try {
    const response = consumer.send('Hello from consumer', 5000);
    console.log('Consumer received response:', response);
  } catch (error) {
    console.error('Consumer send error:', error);
  }

  // Cleanup after response
  setTimeout(() => {
    console.log('Cleaning up...');
    consumer.close();
    provider.close();
    server.stop();
    console.log('Done');
    process.exit(0);
  }, 100);
}, 100);

console.log('Starting provider polling...');
let pollCount = 0;
// Poll for messages
const pollInterval = setInterval(() => {
  pollCount++;
  if (pollCount % 1000 === 0) {
    console.log('Poll attempt:', pollCount);
  }
  
  try {
    const req = provider.receive();
    if (req) {
      console.log('Provider received:', req);
      console.log('Provider responding...');
      provider.respond(req.requestID, `Echo: ${req.data}`);
      clearInterval(pollInterval);
    }
  } catch (error) {
    console.error('Provider receive error:', error);
    clearInterval(pollInterval);
  }
}, 1);

console.log('Setup complete, waiting for events...');
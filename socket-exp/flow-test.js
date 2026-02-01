const { Consumer, Provider, Server } = require('./socket');
const { v4: uuidv4 } = require('uuid');

console.log('Testing request/response flow...');

let server, consumer, provider;
try {
  console.log('1. Starting server...');
  server = new Server('9095');
  
  setTimeout(() => {
    const channelID = uuidv4();
    console.log('2. Creating provider...');
    provider = new Provider('127.0.0.1:9095', channelID);
    
    console.log('3. Creating consumer...');
    consumer = new Consumer('127.0.0.1:9095', channelID);
    
    console.log('4. Sending request...');
    const response = consumer.send('test message', 2000);
    console.log('5. Response received:', response);
    
    consumer.close();
    provider.close();
    server.stop();
    console.log('✓ SUCCESS');
    process.exit(0);
    
  }, 500);
  
} catch (err) {
  console.error('✗ FAIL:', err.message);
  if (consumer) consumer.close();
  if (provider) provider.close();
  if (server) server.stop();
  process.exit(1);
}

setTimeout(() => {
  console.error('✗ TIMEOUT - likely hanging on send');
  if (consumer) consumer.close();
  if (provider) provider.close();
  if (server) server.stop();
  process.exit(1);
}, 5000);
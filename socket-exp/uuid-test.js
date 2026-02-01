const { Consumer, Provider, Server } = require('./socket');
const { v4: uuidv4 } = require('uuid');

console.log('Testing with proper UUID...');

let server;
try {
  console.log('1. Starting server on port 9094...');
  server = new Server('9094');
  console.log('   Server started with ID:', server.id);
  
  setTimeout(() => {
    try {
      const channelID = uuidv4();
      console.log('2. Using channel ID:', channelID);
      
      console.log('3. Creating consumer...');
      const consumer = new Consumer('127.0.0.1:9094', channelID);
      console.log('   Consumer created with ID:', consumer.id);
      
      consumer.close();
      server.stop();
      console.log('✓ SUCCESS');
      process.exit(0);
      
    } catch (err) {
      console.error('✗ FAIL:', err.message);
      if (server) server.stop();
      process.exit(1);
    }
  }, 500);
  
} catch (err) {
  console.error('✗ FAIL:', err.message);
  process.exit(1);
}

setTimeout(() => {
  console.error('✗ TIMEOUT');
  if (server) server.stop();
  process.exit(1);
}, 5000);
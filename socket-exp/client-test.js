const { Consumer, Provider, Server } = require('./socket');

console.log('Testing consumer/provider creation...');

let server;
try {
  // Start server first
  console.log('1. Starting server on port 9093...');
  server = new Server('9093');
  console.log('   Server started with ID:', server.id);
  
  // Wait for server to be ready
  setTimeout(() => {
    try {
      console.log('2. Creating consumer...');
      const consumer = new Consumer('127.0.0.1:9093', 'test-channel-123');
      console.log('   Consumer created with ID:', consumer.id);
      
      console.log('3. Creating provider...');
      const provider = new Provider('127.0.0.1:9093', 'test-channel-123');
      console.log('   Provider created with ID:', provider.id);
      
      // Clean up
      console.log('4. Closing consumer...');
      consumer.close();
      console.log('   Consumer closed, ID now:', consumer.id);
      
      console.log('5. Closing provider...');
      provider.close();
      console.log('   Provider closed, ID now:', provider.id);
      
      console.log('6. Stopping server...');
      server.stop();
      console.log('   Server stopped, ID now:', server.id);
      
      console.log('✓ SUCCESS: All operations completed');
      process.exit(0);
      
    } catch (err) {
      console.error('✗ FAIL: Error with consumer/provider:', err.message);
      if (server) server.stop();
      process.exit(1);
    }
  }, 500);
  
} catch (err) {
  console.error('✗ FAIL: Error starting server:', err.message);
  process.exit(1);
}

// Timeout safety
setTimeout(() => {
  console.error('✗ TIMEOUT: Test took too long, likely hanging');
  if (server) server.stop();
  process.exit(1);
}, 10000);
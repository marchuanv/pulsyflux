const { Server } = require('./socket');

console.log('Testing server start/stop...');

try {
  // Create server
  console.log('1. Creating server on port 9092...');
  const server = new Server('9092');
  console.log('   Server created with ID:', server.id);
  
  if (server.id < 0) {
    throw new Error('Server creation failed');
  }
  
  // Wait a moment
  console.log('2. Waiting 1 second...');
  setTimeout(() => {
    try {
      // Stop server
      console.log('3. Stopping server...');
      server.stop();
      console.log('   Server stopped, ID now:', server.id);
      
      // Verify it's stopped
      if (server.id === -1) {
        console.log('✓ SUCCESS: Server properly stopped');
      } else {
        console.log('✗ FAIL: Server ID not reset to -1');
      }
      
      console.log('Test completed');
      process.exit(0);
    } catch (err) {
      console.error('✗ FAIL: Error stopping server:', err.message);
      process.exit(1);
    }
  }, 1000);
  
} catch (err) {
  console.error('✗ FAIL: Error creating server:', err.message);
  process.exit(1);
}
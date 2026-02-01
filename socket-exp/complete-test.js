const { Consumer, Provider, Server } = require('./socket');
const { v4: uuidv4 } = require('uuid');

console.log('Testing with provider handling...');

let server, consumer, provider;
let requestHandled = false;

try {
  console.log('1. Starting server...');
  server = new Server('9096');
  
  setTimeout(() => {
    const channelID = uuidv4();
    
    console.log('2. Creating provider...');
    provider = new Provider('127.0.0.1:9096', channelID);
    
    // Start provider request handling
    const handleRequests = () => {
      const req = provider.receive();
      if (req && !requestHandled) {
        console.log('4. Provider received:', req.data);
        provider.respond(req.requestID, `Echo: ${req.data}`);
        requestHandled = true;
      }
      if (!requestHandled) {
        setTimeout(handleRequests, 50);
      }
    };
    setTimeout(handleRequests, 100);
    
    console.log('3. Creating consumer...');
    consumer = new Consumer('127.0.0.1:9096', channelID);
    
    setTimeout(() => {
      console.log('5. Sending request...');
      const response = consumer.send('hello world', 3000);
      console.log('6. Response received:', response);
      
      consumer.close();
      provider.close();
      server.stop();
      console.log('✓ SUCCESS: Full flow completed');
      process.exit(0);
    }, 200);
    
  }, 500);
  
} catch (err) {
  console.error('✗ FAIL:', err.message);
  if (consumer) consumer.close();
  if (provider) provider.close();
  if (server) server.stop();
  process.exit(1);
}

setTimeout(() => {
  console.error('✗ TIMEOUT');
  if (consumer) consumer.close();
  if (provider) provider.close();
  if (server) server.stop();
  process.exit(1);
}, 8000);
const { Consumer, Provider } = require('./socket');
const { v4: uuidv4 } = require('uuid');

// Example: Consumer usage
async function consumerExample() {
  const channelID = uuidv4();
  const consumer = new Consumer('127.0.0.1:9090', channelID);

  try {
    const response = consumer.send('Hello from Node.js!', 5000);
    console.log('Response:', response);
  } finally {
    consumer.close();
  }
}

// Example: Provider usage
async function providerExample() {
  const channelID = uuidv4();
  const provider = new Provider('127.0.0.1:9090', channelID);

  try {
    // Listen for requests
    setInterval(() => {
      const req = provider.receive();
      if (req) {
        console.log('Received request:', req.requestID, req.data);
        
        // Send response
        const responseData = `Processed: ${req.data}`;
        provider.respond(req.requestID, responseData);
      }
    }, 100);
  } catch (err) {
    console.error('Provider error:', err);
    provider.close();
  }
}

// Run examples
// consumerExample();
// providerExample();

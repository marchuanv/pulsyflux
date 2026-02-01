const { Consumer, Provider, Server } = require('./socket');
const { v4: uuidv4 } = require('uuid');

describe('Socket Library - Fast', () => {
  let server;
  
  beforeAll(async () => {
    server = new Server('9092');
    // Give server time to start
    await new Promise(resolve => setTimeout(resolve, 100));
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
  });

  it('should create consumer and provider', () => {
    const channelID = uuidv4();
    const consumer = new Consumer('127.0.0.1:9092', channelID);
    const provider = new Provider('127.0.0.1:9092', channelID);
    
    expect(consumer.id).toBeGreaterThanOrEqual(0);
    expect(provider.id).toBeGreaterThanOrEqual(0);
    
    consumer.close();
    provider.close();
  });

  it('should handle basic communication', (done) => {
    const channelID = uuidv4();
    const provider = new Provider('127.0.0.1:9092', channelID);
    const consumer = new Consumer('127.0.0.1:9092', channelID);
    
    // Simple polling instead of continuous interval
    const checkForRequest = () => {
      const req = provider.receive();
      if (req) {
        provider.respond(req.requestID, 'pong');
        consumer.close();
        provider.close();
        done();
      } else {
        setTimeout(checkForRequest, 10);
      }
    };
    
    setTimeout(() => {
      const response = consumer.send('ping', 2000);
      expect(response).toBe('pong');
    }, 50);
    
    checkForRequest();
  }, 3000);
});
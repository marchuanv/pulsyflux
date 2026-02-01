const { Consumer, Provider, Server } = require('./socket');
const { v4: uuidv4 } = require('uuid');

describe('Socket Library', () => {
  const SERVER_ADDR = '127.0.0.1:9090';
  let server;
  let channelID;
  let provider;
  let consumer;

  beforeAll(() => {
    server = new Server('9090');
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
  });

  beforeEach(() => {
    channelID = uuidv4();
  });

  afterEach(() => {
    if (consumer) {
      consumer.close();
      consumer = null;
    }
    if (provider) {
      provider.close();
      provider = null;
    }
  });

  describe('Consumer', () => {
    it('should create a consumer', () => {
      consumer = new Consumer(SERVER_ADDR, channelID);
      expect(consumer).toBeDefined();
      expect(consumer.id).toBeGreaterThanOrEqual(0);
    });

    it('should send and receive a message', (done) => {
      provider = new Provider(SERVER_ADDR, channelID);
      consumer = new Consumer(SERVER_ADDR, channelID);
      
      const interval = setInterval(() => {
        const req = provider.receive();
        if (req) {
          provider.respond(req.requestID, 'pong');
          clearInterval(interval);
        }
      }, 10);

      setTimeout(() => {
        const response = consumer.send('ping', 5000);
        expect(response).toBe('pong');
        clearInterval(interval);
        done();
      }, 100);
    }, 10000);

    it('should handle timeout', (done) => {
      consumer = new Consumer(SERVER_ADDR, channelID);
      
      try {
        consumer.send('test', 100);
        done.fail('Should have thrown');
      } catch (e) {
        expect(e).toBeDefined();
        done();
      }
    }, 5000);
  });

  describe('Provider', () => {
    it('should create a provider', () => {
      provider = new Provider(SERVER_ADDR, channelID);
      expect(provider).toBeDefined();
      expect(provider.id).toBeGreaterThanOrEqual(0);
    });

    it('should receive and respond to requests', (done) => {
      provider = new Provider(SERVER_ADDR, channelID);
      consumer = new Consumer(SERVER_ADDR, channelID);

      setTimeout(() => {
        const req = provider.receive();
        if (req) {
          expect(req.requestID).toBeDefined();
          expect(req.data).toBe('hello');
          provider.respond(req.requestID, 'world');
        }
      }, 100);

      setTimeout(() => {
        const response = consumer.send('hello', 5000);
        expect(response).toBe('world');
        done();
      }, 200);
    }, 10000);

    it('should handle multiple requests', (done) => {
      provider = new Provider(SERVER_ADDR, channelID);
      consumer = new Consumer(SERVER_ADDR, channelID);

      let count = 0;
      const interval = setInterval(() => {
        const req = provider.receive();
        if (req) {
          count++;
          provider.respond(req.requestID, `response-${count}`);
        }
      }, 50);

      setTimeout(() => {
        const r1 = consumer.send('req1', 5000);
        const r2 = consumer.send('req2', 5000);
        const r3 = consumer.send('req3', 5000);

        expect(r1).toContain('response-');
        expect(r2).toContain('response-');
        expect(r3).toContain('response-');
        
        clearInterval(interval);
        done();
      }, 200);
    }, 10000);

    it('should respond with error', (done) => {
      provider = new Provider(SERVER_ADDR, channelID);
      consumer = new Consumer(SERVER_ADDR, channelID);

      const interval = setInterval(() => {
        const req = provider.receive();
        if (req) {
          provider.respond(req.requestID, null, 'test error');
          clearInterval(interval);
        }
      }, 10);

      setTimeout(() => {
        try {
          consumer.send('test', 5000);
          done.fail('Should have thrown');
        } catch (e) {
          expect(e).toBeDefined();
          clearInterval(interval);
          done();
        }
      }, 100);
    }, 10000);
  });

  describe('Multiple Consumers', () => {
    it('should handle multiple consumers on same channel', (done) => {
      provider = new Provider(SERVER_ADDR, channelID);
      const consumer1 = new Consumer(SERVER_ADDR, channelID);
      const consumer2 = new Consumer(SERVER_ADDR, channelID);

      const interval = setInterval(() => {
        const req = provider.receive();
        if (req) {
          provider.respond(req.requestID, `echo: ${req.data}`);
        }
      }, 50);

      setTimeout(() => {
        const r1 = consumer1.send('msg1', 5000);
        const r2 = consumer2.send('msg2', 5000);

        expect(r1).toBe('echo: msg1');
        expect(r2).toBe('echo: msg2');

        consumer1.close();
        consumer2.close();
        clearInterval(interval);
        done();
      }, 200);
    }, 10000);
  });

  describe('Large Payloads', () => {
    it('should handle large messages', (done) => {
      provider = new Provider(SERVER_ADDR, channelID);
      consumer = new Consumer(SERVER_ADDR, channelID);

      const largeData = 'X'.repeat(100000); // 100KB

      setTimeout(() => {
        const req = provider.receive();
        if (req) {
          expect(req.data.length).toBe(100000);
          provider.respond(req.requestID, req.data);
        }
      }, 100);

      setTimeout(() => {
        const response = consumer.send(largeData, 10000);
        expect(response.length).toBe(100000);
        done();
      }, 200);
    }, 15000);
  });
});

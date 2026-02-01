import { Consumer, Provider, Server } from './socket.mjs';
import { v4 as uuidv4 } from 'uuid';

describe('Socket Library', () => {
  const SERVER_ADDR = '127.0.0.1:9090';
  let server;
  let channelID;
  let provider;
  let consumer;
  let intervals = [];

  beforeAll(() => {
    server = new Server('9090');
    if (server) {
      server.start();
    }
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
  });

  beforeEach(() => {
    channelID = uuidv4();
    intervals = [];
  });

  afterEach(() => {
    intervals.forEach(clearInterval);
    intervals = [];
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
          done();
        }
      }, 10);
      intervals.push(interval);

      setTimeout(() => {
        const response = consumer.send('ping', 5000);
        expect(response).toBe('pong');
      }, 100);
    }, 5000);

    it('should handle timeout', (done) => {
      consumer = new Consumer(SERVER_ADDR, channelID);

      try {
        consumer.send('test', 100);
        done.fail('Should have thrown');
      } catch (e) {
        expect(e).toBeDefined();
        done();
      }
    }, 3000);
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

      const interval = setInterval(() => {
        const req = provider.receive();
        if (req) {
          expect(req.requestID).toBeDefined();
          expect(req.data).toBe('hello');
          provider.respond(req.requestID, 'world');
          clearInterval(interval);
        }
      }, 10);
      intervals.push(interval);

      setTimeout(() => {
        const response = consumer.send('hello', 5000);
        expect(response).toBe('world');
        done();
      }, 100);
    }, 5000);

    it('should handle multiple requests', (done) => {
      provider = new Provider(SERVER_ADDR, channelID);
      consumer = new Consumer(SERVER_ADDR, channelID);

      let count = 0;
      const interval = setInterval(() => {
        const req = provider.receive();
        if (req) {
          count++;
          provider.respond(req.requestID, `response-${count}`);
          if (count >= 3) {
            clearInterval(interval);
          }
        }
      }, 10);
      intervals.push(interval);

      setTimeout(() => {
        const r1 = consumer.send('req1', 3000);
        const r2 = consumer.send('req2', 3000);
        const r3 = consumer.send('req3', 3000);

        expect(r1).toContain('response-');
        expect(r2).toContain('response-');
        expect(r3).toContain('response-');

        done();
      }, 100);
    }, 5000);

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
      intervals.push(interval);

      setTimeout(() => {
        try {
          consumer.send('test', 3000);
          done.fail('Should have thrown');
        } catch (e) {
          expect(e).toBeDefined();
          done();
        }
      }, 100);
    }, 5000);
  });

  describe('Multiple Consumers', () => {
    it('should handle multiple consumers on same channel', (done) => {
      provider = new Provider(SERVER_ADDR, channelID);
      const consumer1 = new Consumer(SERVER_ADDR, channelID);
      const consumer2 = new Consumer(SERVER_ADDR, channelID);

      let responses = 0;
      const interval = setInterval(() => {
        const req = provider.receive();
        if (req) {
          provider.respond(req.requestID, `echo: ${req.data}`);
          responses++;
          if (responses >= 2) {
            clearInterval(interval);
          }
        }
      }, 10);
      intervals.push(interval);

      setTimeout(() => {
        const r1 = consumer1.send('msg1', 3000);
        const r2 = consumer2.send('msg2', 3000);

        expect(r1).toBe('echo: msg1');
        expect(r2).toBe('echo: msg2');

        consumer1.close();
        consumer2.close();
        done();
      }, 100);
    }, 5000);
  });

  describe('Large Payloads', () => {
    it('should handle large messages', (done) => {
      provider = new Provider(SERVER_ADDR, channelID);
      consumer = new Consumer(SERVER_ADDR, channelID);

      const largeData = 'X'.repeat(10000); // 10KB (smaller for faster test)

      const interval = setInterval(() => {
        const req = provider.receive();
        if (req) {
          expect(req.data.length).toBe(10000);
          provider.respond(req.requestID, req.data);
          clearInterval(interval);
        }
      }, 10);
      intervals.push(interval);

      setTimeout(() => {
        const response = consumer.send(largeData, 5000);
        expect(response.length).toBe(10000);
        done();
      }, 100);
    }, 8000);
  });
});

import { Consumer, Provider, Server } from '../wrapper.mjs';
import { v4 as uuidv4 } from 'uuid';

describe('Socket Library', () => {
  const SERVER_ADDR = '127.0.0.1:9090';
  let server;
  let channelID;
  let provider;
  let consumer;

  beforeAll(async () => {
    server = new Server('9090');
    await server.start();
    await new Promise(resolve => setTimeout(resolve, 100));
  });

  afterAll(async () => {
    await server?.stop();
  });

  beforeEach(() => {
    channelID = uuidv4();
  });

  afterEach(async () => {
    await consumer?.close();
    await provider?.close();
    consumer = null;
    provider = null;
  });

  describe('Consumer', () => {
    it('should create a consumer', () => {
      consumer = new Consumer(SERVER_ADDR, channelID);
      expect(consumer).toBeDefined();
    });

    it('should send and receive a message', async () => {
      provider = new Provider(SERVER_ADDR, channelID);
      consumer = new Consumer(SERVER_ADDR, channelID);

      const responsePromise = consumer.send('ping', 5000);
      const req = await provider.receive();
      await provider.respond(req.requestID, 'pong');
      const response = await responsePromise;

      expect(response).toBe('pong');
    }, 40000);

    it('should handle timeout', async () => {
      consumer = new Consumer(SERVER_ADDR, channelID);
      const start = Date.now();
      try {
        await consumer.send('test', 500); // Increase timeout to 500ms
        fail('Should have thrown');
      } catch (error) {
        const elapsed = Date.now() - start;
        expect(error).toBeDefined();
        expect(elapsed).toBeGreaterThan(400); // Should timeout around 500ms
        expect(elapsed).toBeLessThan(1000); // But not take too long
      }
    }, 10000);
  });

  describe('Provider', () => {
    it('should create a provider', () => {
      provider = new Provider(SERVER_ADDR, channelID);
      expect(provider).toBeDefined();
    });

    it('should receive and respond to requests', async () => {
      provider = new Provider(SERVER_ADDR, channelID);
      consumer = new Consumer(SERVER_ADDR, channelID);

      const responsePromise = consumer.send('hello', 5000);
      const req = await provider.receive();

      expect(req.requestID).toBeDefined();
      expect(req.data).toBe('hello');

      await provider.respond(req.requestID, 'world');
      const response = await responsePromise;

      expect(response).toBe('world');
    });

    it('should handle multiple requests', async () => {
      provider = new Provider(SERVER_ADDR, channelID);
      consumer = new Consumer(SERVER_ADDR, channelID);

      const promises = [
        consumer.send('req1', 3000),
        consumer.send('req2', 3000),
        consumer.send('req3', 3000)
      ];

      for (let i = 0; i < 3; i++) {
        const req = await provider.receive();
        await provider.respond(req.requestID, `response-${i + 1}`);
      }

      const responses = await Promise.all(promises);
      responses.forEach(r => expect(r).toContain('response-'));
    });

    it('should respond with error', async () => {
      provider = new Provider(SERVER_ADDR, channelID);
      consumer = new Consumer(SERVER_ADDR, channelID);

      const responsePromise = consumer.send('test', 3000);
      const req = await provider.receive();
      await provider.respond(req.requestID, null, 'test error');

      try {
        await responsePromise;
        fail('Should have thrown');
      } catch (error) {
        expect(error).toBeDefined();
      }
    });
  });

  describe('Multiple Consumers', () => {
    it('should handle multiple consumers on same channel', async () => {
      provider = new Provider(SERVER_ADDR, channelID);
      const consumer1 = new Consumer(SERVER_ADDR, channelID);
      const consumer2 = new Consumer(SERVER_ADDR, channelID);

      const promises = [
        consumer1.send('msg1', 3000),
        consumer2.send('msg2', 3000)
      ];

      for (let i = 0; i < 2; i++) {
        const req = await provider.receive();
        await provider.respond(req.requestID, `echo: ${req.data}`);
      }

      const [r1, r2] = await Promise.all(promises);
      expect(r1).toBe('echo: msg1');
      expect(r2).toBe('echo: msg2');

      await consumer1.close();
      await consumer2.close();
    });
  });

  describe('Large Payloads', () => {
    it('should handle large messages', async () => {
      provider = new Provider(SERVER_ADDR, channelID);
      consumer = new Consumer(SERVER_ADDR, channelID);

      const largeData = 'X'.repeat(10000);
      const responsePromise = consumer.send(largeData, 5000);

      const req = await provider.receive();
      expect(req.data.length).toBe(10000);
      await provider.respond(req.requestID, req.data);

      const response = await responsePromise;
      expect(response.length).toBe(10000);
    });
  });
});

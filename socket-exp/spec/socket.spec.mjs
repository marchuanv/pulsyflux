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
    try {
      await server?.stop();
    } catch (error) {
      // Ignore cleanup errors
    }
    // Brief wait for cleanup
    await new Promise(resolve => setTimeout(resolve, 50));
    // Force exit after tests complete
    setTimeout(() => {
      console.log('Forcing process exit...');
      process.exit(0);
    }, 500);
  });

  beforeEach(() => {
    channelID = uuidv4();
  });

  afterEach(async () => {
    try {
      if (consumer) {
        await consumer.close();
        consumer = null;
      }
    } catch (error) {
      // Ignore cleanup errors
    }
    try {
      if (provider) {
        await provider.close();
        provider = null;
      }
    } catch (error) {
      // Ignore cleanup errors
    }
    // Brief wait for cleanup to complete
    await new Promise(resolve => setTimeout(resolve, 10));
  });

  describe('Consumer', () => {
    it('should create a consumer', async () => {
      consumer = new Consumer(SERVER_ADDR, channelID);
      expect(consumer).toBeDefined();
      await new Promise(resolve => setTimeout(resolve, 10));
    });

    it('should send and receive a message', async () => {
      provider = new Provider(SERVER_ADDR, channelID);
      await new Promise(resolve => setTimeout(resolve, 200)); // Wait for provider polling to start
      consumer = new Consumer(SERVER_ADDR, channelID);
      await new Promise(resolve => setTimeout(resolve, 100)); // Wait for consumer to connect

      const responsePromise = consumer.send('ping', 5000);
      const req = await provider.receive();
      await provider.respond(req.requestID, 'pong');
      const response = await responsePromise;

      expect(response).toBe('pong');
    });

    it('should handle timeout', async () => {
      consumer = new Consumer(SERVER_ADDR, channelID);
      await new Promise(resolve => setTimeout(resolve, 100)); // Wait for consumer to connect
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
    });
  });

  describe('Provider', () => {
    it('should create a provider', async () => {
      provider = new Provider(SERVER_ADDR, channelID);
      expect(provider).toBeDefined();
      await new Promise(resolve => setTimeout(resolve, 100)); // Wait for provider to start
    });

    it('should receive and respond to requests', async () => {
      provider = new Provider(SERVER_ADDR, channelID);
      await new Promise(resolve => setTimeout(resolve, 200)); // Wait for provider polling to start
      consumer = new Consumer(SERVER_ADDR, channelID);
      await new Promise(resolve => setTimeout(resolve, 100)); // Wait for consumer to connect

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
      await new Promise(resolve => setTimeout(resolve, 200)); // Wait for provider polling to start
      consumer = new Consumer(SERVER_ADDR, channelID);
      await new Promise(resolve => setTimeout(resolve, 100)); // Wait for consumer to connect

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
      await new Promise(resolve => setTimeout(resolve, 200)); // Wait for provider polling to start
      consumer = new Consumer(SERVER_ADDR, channelID);
      await new Promise(resolve => setTimeout(resolve, 100)); // Wait for consumer to connect

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
      await new Promise(resolve => setTimeout(resolve, 200)); // Wait for provider polling to start
      const consumer1 = new Consumer(SERVER_ADDR, channelID);
      const consumer2 = new Consumer(SERVER_ADDR, channelID);
      await new Promise(resolve => setTimeout(resolve, 100)); // Wait for consumers to connect

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
      await new Promise(resolve => setTimeout(resolve, 200)); // Wait for provider polling to start
      consumer = new Consumer(SERVER_ADDR, channelID);
      await new Promise(resolve => setTimeout(resolve, 100)); // Wait for consumer to connect

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

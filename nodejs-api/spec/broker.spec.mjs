import { Server, Client } from '../registry.mjs';
import { randomUUID } from 'crypto';
import { createRequire } from 'module';
const require = createRequire(import.meta.url);
const addon = require('../broker_addon.node');

describe('Broker', () => {
  let server;
  let client1;
  let client2;
  let channelID;

  beforeAll(() => {
    server = new Server(':0');
    server.start();
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
    if (addon.cleanup) {
      addon.cleanup();
    }
  });

  beforeEach(() => {
    channelID = randomUUID();
    client1 = new Client(server.addr(), channelID);
    client2 = new Client(server.addr(), channelID);
  });

  afterEach(() => {
    if (client1) {
      client1.close();
      client1 = null;
    }
    if (client2) {
      client2.close();
      client2 = null;
    }
  });

  describe('Server', () => {
    it('should start and get address', () => {
      const testServer = new Server(':0');
      expect(() => testServer.start()).not.toThrow();
      expect(testServer.addr()).toBeTruthy();
      expect(() => testServer.stop()).not.toThrow();
    });
  });

  describe('Client', () => {
    it('should create a client', () => {
      expect(client1).toBeDefined();
    });

    it('should verify client methods exist', () => {
      expect(typeof client1.publish).toBe('function');
      expect(typeof client1.subscribe).toBe('function');
    });

    it('should publish and receive messages', (done) => {
      client2.onMessage((msg) => {
        expect(msg.toString()).toBe('Hello World');
        done();
      });

      client1.publish('Hello World');
    });

    it('should handle binary payloads', (done) => {
      const data = Buffer.from([1, 2, 3, 4, 5]);

      client2.onMessage((msg) => {
        expect(Buffer.compare(msg, data)).toBe(0);
        done();
      });

      client1.publish(data);
    });

    it('should handle JSON payloads', (done) => {
      const data = { id: 123, name: 'test' };

      client2.onMessage((msg) => {
        const received = JSON.parse(msg.toString());
        expect(received.id).toBe(123);
        expect(received.name).toBe('test');
        done();
      });

      client1.publish(JSON.stringify(data));
    });

    it('should handle multiple messages', (done) => {
      const messages = [];

      client2.onMessage((msg) => {
        messages.push(msg.toString());
        if (messages.length === 3) {
          expect(messages).toContain('msg1');
          expect(messages).toContain('msg2');
          expect(messages).toContain('msg3');
          done();
        }
      });

      client1.publish('msg1');
      client1.publish('msg2');
      client1.publish('msg3');
    });

    it('should not receive own messages', (done) => {
      client1.onMessage(() => {
        fail('Client1 should not receive own message');
      });

      client2.onMessage(() => {
        expect(true).toBe(true);
        done();
      });

      client1.publish('test');
    });
  });

  describe('Subscription', () => {
    it('should return null when no messages', () => {
      const msg = client1.subscribe();
      expect(msg).toBeNull();
    });
  });

  describe('Channel Isolation', () => {
    it('should isolate messages between channels', (done) => {
      const channel1 = randomUUID();
      const channel2 = randomUUID();

      const c1 = new Client(server.addr(), channel1);
      const c2 = new Client(server.addr(), channel1);
      const c3 = new Client(server.addr(), channel2);
      const c4 = new Client(server.addr(), channel2);

      let msg2 = null;
      let msg4 = null;

      c2.onMessage((msg) => {
        msg2 = msg;
        checkComplete();
      });

      c4.onMessage((msg) => {
        msg4 = msg;
        checkComplete();
      });

      function checkComplete() {
        if (msg2 && msg4) {
          expect(msg2.toString()).toBe('channel1 message');
          expect(msg4.toString()).toBe('channel2 message');
          c1.close();
          c2.close();
          c3.close();
          c4.close();
          done();
        }
      }

      c1.publish('channel1 message');
      c3.publish('channel2 message');
    });
  });
});
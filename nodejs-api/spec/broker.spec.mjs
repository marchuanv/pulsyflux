import { Server, Client } from '../broker.mjs';
import { randomUUID } from 'crypto';

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
  });

  beforeEach(() => {
    channelID = randomUUID();
    client1 = new Client(server.addr(), channelID);
    client2 = new Client(server.addr(), channelID);
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
      expect(client1.id).toBeGreaterThanOrEqual(0);
    });

    it('should publish and receive messages', (done) => {
      const sub = client2.subscribe();
      
      setTimeout(() => client1.publish('Hello World'), 50);

      const poll = setInterval(() => {
        const msg = sub.receive();
        if (msg) {
          clearInterval(poll);
          expect(msg.toString()).toBe('Hello World');
          sub.close();
          done();
        }
      }, 10);

      setTimeout(() => {
        clearInterval(poll);
        fail('Timeout waiting for message');
      }, 2000);
    }, 5000);

    it('should handle binary payloads', (done) => {
      const sub = client2.subscribe();
      const data = Buffer.from([1, 2, 3, 4, 5]);
      
      setTimeout(() => client1.publish(data), 50);

      const poll = setInterval(() => {
        const msg = sub.receive();
        if (msg) {
          clearInterval(poll);
          expect(Buffer.compare(msg, data)).toBe(0);
          sub.close();
          done();
        }
      }, 10);

      setTimeout(() => {
        clearInterval(poll);
        fail('Timeout waiting for message');
      }, 2000);
    }, 5000);

    it('should handle JSON payloads', (done) => {
      const sub = client2.subscribe();
      const data = { id: 123, name: 'test' };
      
      setTimeout(() => client1.publish(JSON.stringify(data)), 50);

      const poll = setInterval(() => {
        const msg = sub.receive();
        if (msg) {
          clearInterval(poll);
          const received = JSON.parse(msg.toString());
          expect(received.id).toBe(123);
          expect(received.name).toBe('test');
          sub.close();
          done();
        }
      }, 10);

      setTimeout(() => {
        clearInterval(poll);
        fail('Timeout waiting for message');
      }, 2000);
    }, 5000);

    it('should handle async iteration', async () => {
      const sub = client2.subscribe();
      
      setTimeout(() => client1.publish('async test'), 50);

      for await (const msg of sub) {
        expect(msg.toString()).toBe('async test');
        sub.close();
        break;
      }
    }, 5000);

    it('should handle multiple messages', (done) => {
      const sub = client2.subscribe();
      const messages = [];
      
      setTimeout(() => {
        client1.publish('msg1');
        client1.publish('msg2');
        client1.publish('msg3');
      }, 50);

      const poll = setInterval(() => {
        const msg = sub.receive();
        if (msg) {
          messages.push(msg.toString());
        }
        
        if (messages.length === 3) {
          clearInterval(poll);
          expect(messages).toContain('msg1');
          expect(messages).toContain('msg2');
          expect(messages).toContain('msg3');
          sub.close();
          done();
        }
      }, 10);

      setTimeout(() => {
        clearInterval(poll);
        fail(`Only received ${messages.length}/3 messages`);
      }, 2000);
    }, 5000);

    it('should not receive own messages', (done) => {
      const sub1 = client1.subscribe();
      const sub2 = client2.subscribe();
      
      setTimeout(() => client1.publish('test'), 50);

      let client2Received = false;
      const poll = setInterval(() => {
        const msg1 = sub1.receive();
        const msg2 = sub2.receive();
        
        if (msg1) {
          clearInterval(poll);
          fail('Client1 should not receive own message');
        }
        
        if (msg2) {
          client2Received = true;
        }
      }, 10);

      setTimeout(() => {
        clearInterval(poll);
        expect(client2Received).toBe(true);
        sub1.close();
        sub2.close();
        done();
      }, 500);
    }, 5000);
  });

  describe('Subscription', () => {
    it('should return null when no messages', () => {
      const sub = client1.subscribe();
      const msg = sub.receive();
      expect(msg).toBeNull();
      sub.close();
    });

    it('should close cleanly', () => {
      const sub = client1.subscribe();
      expect(() => sub.close()).not.toThrow();
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
      
      const sub2 = c2.subscribe();
      const sub4 = c4.subscribe();
      
      setTimeout(() => {
        c1.publish('channel1 message');
        c3.publish('channel2 message');
      }, 50);

      let msg2 = null;
      let msg4 = null;
      
      const poll = setInterval(() => {
        if (!msg2) msg2 = sub2.receive();
        if (!msg4) msg4 = sub4.receive();
        
        if (msg2 && msg4) {
          clearInterval(poll);
          expect(msg2.toString()).toBe('channel1 message');
          expect(msg4.toString()).toBe('channel2 message');
          sub2.close();
          sub4.close();
          done();
        }
      }, 10);

      setTimeout(() => {
        clearInterval(poll);
        fail('Timeout waiting for messages');
      }, 2000);
    }, 5000);
  });
});

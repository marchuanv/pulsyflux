import { Server, Bus } from '../messagebus.mjs';
import { v4 as uuidv4 } from 'uuid';

describe('MessageBus', () => {
  let server;
  let bus;
  const channelID = uuidv4();

  beforeAll(() => {
    server = new Server('9090');
    server.start();
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
  });

  beforeEach(() => {
    bus = new Bus('127.0.0.1:9090', channelID);
  });

  afterEach(() => {
    if (bus) {
      bus.close();
    }
  });

  describe('Server', () => {
    it('should start and stop', () => {
      const testServer = new Server('9091');
      expect(() => testServer.start()).not.toThrow();
      expect(() => testServer.stop()).not.toThrow();
    });
  });

  describe('Bus', () => {
    it('should create a bus', () => {
      expect(bus).toBeDefined();
      expect(bus.id).toBeGreaterThanOrEqual(0);
    });

    it('should publish and receive messages', (done) => {
      const sub = bus.subscribe('test.topic');
      
      setTimeout(() => {
        bus.publish('test.topic', 'Hello World');
      }, 200);

      setTimeout(() => {
        const msg = sub.receive();
        if (msg) {
          expect(msg.topic).toBe('test.topic');
          expect(msg.payload.toString()).toBe('Hello World');
        } else {
          fail('No message received');
        }
        sub.close();
        done();
      }, 400);
    }, 10000);

    it('should handle headers', (done) => {
      const sub = bus.subscribe('test.headers');
      
      setTimeout(() => {
        bus.publish('test.headers', 'data', {
          user_id: '123',
          source: 'test'
        });
      }, 100);

      setTimeout(() => {
        const msg = sub.receive();
        expect(msg).not.toBeNull();
        expect(msg.headers).toBeDefined();
        expect(msg.headers.user_id).toBe('123');
        expect(msg.headers.source).toBe('test');
        sub.close();
        done();
      }, 200);
    }, 5000);

    it('should support multiple subscribers', (done) => {
      const sub1 = bus.subscribe('test.multi');
      const sub2 = bus.subscribe('test.multi');
      
      setTimeout(() => {
        bus.publish('test.multi', 'broadcast');
      }, 100);

      setTimeout(() => {
        const msg1 = sub1.receive();
        const msg2 = sub2.receive();
        
        expect(msg1).not.toBeNull();
        expect(msg2).not.toBeNull();
        expect(msg1.payload.toString()).toBe('broadcast');
        expect(msg2.payload.toString()).toBe('broadcast');
        
        sub1.close();
        sub2.close();
        done();
      }, 200);
    }, 5000);

    it('should handle JSON payloads', (done) => {
      const sub = bus.subscribe('test.json');
      const data = { id: 123, name: 'test' };
      
      setTimeout(() => {
        bus.publish('test.json', JSON.stringify(data));
      }, 100);

      setTimeout(() => {
        const msg = sub.receive();
        expect(msg).not.toBeNull();
        const received = JSON.parse(msg.payload.toString());
        expect(received.id).toBe(123);
        expect(received.name).toBe('test');
        sub.close();
        done();
      }, 200);
    }, 5000);

    it('should handle unsubscribe', (done) => {
      const sub = bus.subscribe('test.unsub');
      
      setTimeout(() => {
        bus.unsubscribe('test.unsub');
        bus.publish('test.unsub', 'should not receive');
      }, 100);

      setTimeout(() => {
        const msg = sub.receive();
        expect(msg).toBeNull();
        sub.close();
        done();
      }, 200);
    }, 5000);

    it('should handle async iteration', (done) => {
      const sub = bus.subscribe('test.async');
      let received = false;

      (async () => {
        for await (const msg of sub) {
          expect(msg.topic).toBe('test.async');
          expect(msg.payload.toString()).toBe('async test');
          received = true;
          sub.close();
          break;
        }
      })();

      setTimeout(() => {
        bus.publish('test.async', 'async test');
      }, 100);

      setTimeout(() => {
        expect(received).toBe(true);
        done();
      }, 500);
    }, 5000);

    it('should handle multiple messages', (done) => {
      const sub = bus.subscribe('test.multiple');
      const messages = [];
      
      setTimeout(() => {
        bus.publish('test.multiple', 'msg1');
        bus.publish('test.multiple', 'msg2');
        bus.publish('test.multiple', 'msg3');
      }, 100);

      const checkMessages = setInterval(() => {
        const msg = sub.receive();
        if (msg) {
          messages.push(msg.payload.toString());
        }
        
        if (messages.length === 3) {
          clearInterval(checkMessages);
          expect(messages).toContain('msg1');
          expect(messages).toContain('msg2');
          expect(messages).toContain('msg3');
          sub.close();
          done();
        }
      }, 50);

      setTimeout(() => {
        clearInterval(checkMessages);
        if (messages.length < 3) {
          fail('Did not receive all messages');
        }
      }, 2000);
    }, 5000);
  });

  describe('Subscription', () => {
    it('should return null when no messages', () => {
      const sub = bus.subscribe('test.empty');
      const msg = sub.receive();
      expect(msg).toBeNull();
      sub.close();
    });

    it('should close cleanly', () => {
      const sub = bus.subscribe('test.close');
      expect(() => sub.close()).not.toThrow();
    });
  });
});

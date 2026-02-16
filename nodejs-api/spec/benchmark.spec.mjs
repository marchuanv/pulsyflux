import { Server, Bus } from '../messagebus.mjs';
import { v4 as uuidv4 } from 'uuid';

const sleep = (ms) => new Promise(resolve => setTimeout(resolve, ms));

describe('MessageBus Performance Benchmarks', () => {
  
  describe('Single Request/Response', () => {
    it('should measure latency and throughput', async () => {
      const server = new Server('9200');
      server.start();
      await sleep(50);
      
      const channelID = uuidv4();
      const bus = new Bus('127.0.0.1:9200', channelID);
      const sub = bus.subscribe('bench.single');
      
      const handler = setInterval(() => {
        const msg = sub.receive();
        if (msg) bus.publish('bench.single.response', msg.payload);
      }, 1);
      
      await sleep(50);
      const responseSub = bus.subscribe('bench.single.response');
      
      const iterations = 1000;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        bus.publish('bench.single', 'test');
        while (true) {
          const resp = responseSub.receive();
          if (resp) break;
          await sleep(0);
        }
      }
      
      const end = process.hrtime.bigint();
      const elapsed = Number(end - start) / 1e6;
      const avgLatency = elapsed / iterations;
      const opsPerSec = (iterations / elapsed) * 1000;
      
      console.log(`\n  Iterations: ${iterations}`);
      console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
      console.log(`  Avg Latency: ${avgLatency.toFixed(3)} ms`);
      console.log(`  Throughput: ${opsPerSec.toFixed(0)} ops/sec`);
      
      clearInterval(handler);
      sub.close();
      responseSub.close();
      bus.close();
      server.stop();
      
      expect(avgLatency).toBeLessThan(20);
    }, 30000);
  });

  describe('Concurrent Publish', () => {
    it('should handle high message volume', async () => {
      const server = new Server('9201');
      server.start();
      await sleep(50);
      
      const channelID = uuidv4();
      const bus = new Bus('127.0.0.1:9201', channelID);
      const sub = bus.subscribe('bench.concurrent');
      
      let received = 0;
      const handler = setInterval(() => {
        const msg = sub.receive();
        if (msg) received++;
      }, 1);
      
      await sleep(50);
      
      const iterations = 1000;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        bus.publish('bench.concurrent', 'test');
      }
      
      const timeout = Date.now() + 50000;
      while (received < iterations && Date.now() < timeout) {
        await sleep(10);
      }
      
      const end = process.hrtime.bigint();
      const elapsed = Number(end - start) / 1e6;
      const opsPerSec = (iterations / elapsed) * 1000;
      
      console.log(`\n  Iterations: ${iterations}`);
      console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
      console.log(`  Throughput: ${opsPerSec.toFixed(0)} ops/sec`);
      
      clearInterval(handler);
      sub.close();
      bus.close();
      server.stop();
      
      expect(received).toBe(iterations);
    }, 60000);
  });

  describe('Large Payload (1MB)', () => {
    it('should handle large messages efficiently', async () => {
      const server = new Server('9202');
      server.start();
      await sleep(50);
      
      const channelID = uuidv4();
      const bus = new Bus('127.0.0.1:9202', channelID);
      const sub = bus.subscribe('bench.large');
      
      const largeData = 'X'.repeat(1024 * 1024);
      
      const handler = setInterval(() => {
        const msg = sub.receive();
        if (msg) bus.publish('bench.large.response', msg.payload);
      }, 1);
      
      await sleep(50);
      const responseSub = bus.subscribe('bench.large.response');
      
      const iterations = 100;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        bus.publish('bench.large', largeData);
        while (true) {
          const resp = responseSub.receive();
          if (resp) break;
          await sleep(0);
        }
      }
      
      const end = process.hrtime.bigint();
      const elapsed = Number(end - start) / 1e6;
      const avgLatency = elapsed / iterations;
      const totalBytes = iterations * largeData.length * 2;
      const mbps = (totalBytes / (elapsed / 1000)) / (1024 * 1024);
      
      console.log(`\n  Iterations: ${iterations}`);
      console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
      console.log(`  Avg Latency: ${avgLatency.toFixed(3)} ms`);
      console.log(`  Bandwidth: ${mbps.toFixed(2)} MB/s`);
      
      clearInterval(handler);
      sub.close();
      responseSub.close();
      bus.close();
      server.stop();
      
      expect(avgLatency).toBeLessThan(100);
    }, 60000);
  });

  describe('Multiple Subscribers', () => {
    it('should broadcast to all subscribers', async () => {
      const server = new Server('9203');
      server.start();
      await sleep(50);
      
      const channelID = uuidv4();
      const bus = new Bus('127.0.0.1:9203', channelID);
      
      const numSubscribers = 10;
      const subscribers = [];
      const counters = [];
      const handlers = [];
      
      for (let i = 0; i < numSubscribers; i++) {
        const sub = bus.subscribe('bench.multi');
        subscribers.push(sub);
        counters.push(0);
        
        const handler = setInterval(((idx) => () => {
          const msg = sub.receive();
          if (msg) counters[idx]++;
        })(i), 1);
        handlers.push(handler);
      }
      
      await sleep(50);
      
      const iterations = 500;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        bus.publish('bench.multi', 'test');
      }
      
      const timeout = Date.now() + 50000;
      while (counters.some(c => c < iterations) && Date.now() < timeout) {
        await sleep(10);
      }
      
      const end = process.hrtime.bigint();
      const elapsed = Number(end - start) / 1e6;
      const totalMessages = iterations * numSubscribers;
      const opsPerSec = (totalMessages / elapsed) * 1000;
      
      console.log(`\n  Subscribers: ${numSubscribers}`);
      console.log(`  Messages: ${iterations}`);
      console.log(`  Total Deliveries: ${totalMessages}`);
      console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
      console.log(`  Throughput: ${opsPerSec.toFixed(0)} deliveries/sec`);
      
      handlers.forEach(h => clearInterval(h));
      subscribers.forEach(s => s.close());
      bus.close();
      server.stop();
      
      counters.forEach(c => expect(c).toBe(iterations));
    }, 60000);
  });

  describe('Throughput Test', () => {
    it('should measure maximum throughput', async () => {
      const server = new Server('9204');
      server.start();
      await sleep(50);
      
      const channelID = uuidv4();
      const bus = new Bus('127.0.0.1:9204', channelID);
      const sub = bus.subscribe('bench.throughput');
      
      let received = 0;
      const handler = setInterval(() => {
        while (true) {
          const msg = sub.receive();
          if (!msg) break;
          received++;
        }
      }, 1);
      
      await sleep(50);
      
      const iterations = 1000;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        bus.publish('bench.throughput', 'test');
      }
      
      const timeout = Date.now() + 50000;
      while (received < iterations && Date.now() < timeout) {
        await sleep(10);
      }
      
      const end = process.hrtime.bigint();
      const elapsed = Number(end - start) / 1e6;
      const opsPerSec = (iterations / elapsed) * 1000;
      
      console.log(`\n  Iterations: ${iterations}`);
      console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
      console.log(`  Throughput: ${opsPerSec.toFixed(0)} ops/sec`);
      
      clearInterval(handler);
      sub.close();
      bus.close();
      server.stop();
      
      expect(received).toBe(iterations);
    }, 60000);
  });

  describe('With Headers', () => {
    it('should handle messages with headers', async () => {
      const server = new Server('9205');
      server.start();
      await sleep(50);
      
      const channelID = uuidv4();
      const bus = new Bus('127.0.0.1:9205', channelID);
      const sub = bus.subscribe('bench.headers');
      
      const handler = setInterval(() => {
        const msg = sub.receive();
        if (msg) bus.publish('bench.headers.response', msg.payload, msg.headers);
      }, 1);
      
      await sleep(50);
      const responseSub = bus.subscribe('bench.headers.response');
      
      const iterations = 1000;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        bus.publish('bench.headers', 'test', {
          user_id: '123',
          request_id: `req-${i}`,
          timestamp: Date.now().toString()
        });
        
        while (true) {
          const resp = responseSub.receive();
          if (resp) break;
          await sleep(0);
        }
      }
      
      const end = process.hrtime.bigint();
      const elapsed = Number(end - start) / 1e6;
      const avgLatency = elapsed / iterations;
      const opsPerSec = (iterations / elapsed) * 1000;
      
      console.log(`\n  Iterations: ${iterations}`);
      console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
      console.log(`  Avg Latency: ${avgLatency.toFixed(3)} ms`);
      console.log(`  Throughput: ${opsPerSec.toFixed(0)} ops/sec`);
      
      clearInterval(handler);
      sub.close();
      responseSub.close();
      bus.close();
      server.stop();
      
      expect(avgLatency).toBeLessThan(20);
    }, 60000);
  });

  describe('Bandwidth Test', () => {
    it('should measure bandwidth with 1MB payloads', async () => {
      const server = new Server('9206');
      server.start();
      await sleep(50);
      
      const channelID = uuidv4();
      const bus = new Bus('127.0.0.1:9206', channelID);
      const sub = bus.subscribe('bench.bandwidth');
      
      const largeData = 'X'.repeat(1024 * 1024);
      
      let received = 0;
      const handler = setInterval(() => {
        const msg = sub.receive();
        if (msg) received++;
      }, 1);
      
      await sleep(50);
      
      const iterations = 50;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        bus.publish('bench.bandwidth', largeData);
      }
      
      while (received < iterations) {
        await sleep(10);
      }
      
      const end = process.hrtime.bigint();
      const elapsed = Number(end - start) / 1e9;
      const totalBytes = iterations * largeData.length;
      const mbps = (totalBytes / elapsed) / (1024 * 1024);
      
      console.log(`\n  Iterations: ${iterations}`);
      console.log(`  Total Time: ${(elapsed * 1000).toFixed(2)} ms`);
      console.log(`  Total Data: ${(totalBytes / (1024 * 1024)).toFixed(2)} MB`);
      console.log(`  Bandwidth: ${mbps.toFixed(2)} MB/s`);
      
      clearInterval(handler);
      sub.close();
      bus.close();
      server.stop();
      
      expect(received).toBe(iterations);
      expect(mbps).toBeGreaterThan(10);
    }, 60000);
  });

  describe('Medium Payload (10KB)', () => {
    it('should handle medium-sized messages', async () => {
      const server = new Server('9207');
      server.start();
      await sleep(50);
      
      const channelID = uuidv4();
      const bus = new Bus('127.0.0.1:9207', channelID);
      const sub = bus.subscribe('bench.medium');
      
      const mediumData = 'X'.repeat(10 * 1024);
      
      const handler = setInterval(() => {
        const msg = sub.receive();
        if (msg) bus.publish('bench.medium.response', msg.payload);
      }, 1);
      
      await sleep(50);
      const responseSub = bus.subscribe('bench.medium.response');
      
      const iterations = 1000;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        bus.publish('bench.medium', mediumData);
        while (true) {
          const resp = responseSub.receive();
          if (resp) break;
          await sleep(0);
        }
      }
      
      const end = process.hrtime.bigint();
      const elapsed = Number(end - start) / 1e6;
      const avgLatency = elapsed / iterations;
      const opsPerSec = (iterations / elapsed) * 1000;
      
      console.log(`\n  Iterations: ${iterations}`);
      console.log(`  Payload Size: 10 KB`);
      console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
      console.log(`  Avg Latency: ${avgLatency.toFixed(3)} ms`);
      console.log(`  Throughput: ${opsPerSec.toFixed(0)} ops/sec`);
      
      clearInterval(handler);
      sub.close();
      responseSub.close();
      bus.close();
      server.stop();
      
      expect(avgLatency).toBeLessThan(20);
    }, 30000);
  });
});

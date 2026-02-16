import { Server, Client } from '../broker.mjs';
import { randomUUID } from 'crypto';

const sleep = (ms) => new Promise(resolve => setTimeout(resolve, ms));

describe('Broker Performance Benchmarks', () => {
  
  describe('Single Request/Response', () => {
    it('should measure latency and throughput', async () => {
      const server = new Server(':0');
      server.start();
      await sleep(50);
      
      const channelID = randomUUID();
      const client1 = new Client(server.addr(), channelID);
      const client2 = new Client(server.addr(), channelID);
      
      const sub1 = client1.subscribe();
      const sub2 = client2.subscribe();
      
      const handler = setInterval(() => {
        const msg = sub2.receive();
        if (msg) client2.publish(msg);
      }, 1);
      
      await sleep(100);
      
      const iterations = 1000;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        client1.publish('test');
        while (true) {
          const resp = sub1.receive();
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
      sub1.close();
      sub2.close();
      server.stop();
      
      expect(avgLatency).toBeLessThan(100);
    }, 30000);
  });

  describe('Publish Throughput', () => {
    it('should handle high message volume', async () => {
      const server = new Server(':0');
      server.start();
      await sleep(50);
      
      const channelID = randomUUID();
      const client1 = new Client(server.addr(), channelID);
      const client2 = new Client(server.addr(), channelID);
      
      const sub = client2.subscribe();
      
      let received = 0;
      const handler = setInterval(() => {
        const msg = sub.receive();
        if (msg) received++;
      }, 1);
      
      await sleep(100);
      
      const iterations = 100;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        client1.publish('test');
      }
      
      const timeout = Date.now() + 10000;
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
      server.stop();
      
      expect(received).toBe(iterations);
    }, 30000);
  });

  describe('Broadcast to Multiple Clients', () => {
    it('should broadcast efficiently', async () => {
      const server = new Server(':0');
      server.start();
      await sleep(50);
      
      const channelID = randomUUID();
      const numClients = 5;
      const clients = [];
      const subs = [];
      const counters = [];
      const handlers = [];
      
      for (let i = 0; i < numClients; i++) {
        const client = new Client(server.addr(), channelID);
        clients.push(client);
        const sub = client.subscribe();
        subs.push(sub);
        counters.push(0);
        
        const handler = setInterval(((idx) => () => {
          const msg = sub.receive();
          if (msg) counters[idx]++;
        })(i), 1);
        handlers.push(handler);
      }
      
      await sleep(100);
      
      const iterations = 100;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        clients[0].publish('test');
      }
      
      const timeout = Date.now() + 10000;
      while (counters.some(c => c < iterations - 1) && Date.now() < timeout) {
        await sleep(10);
      }
      
      const end = process.hrtime.bigint();
      const elapsed = Number(end - start) / 1e6;
      const totalMessages = iterations * (numClients - 1);
      const opsPerSec = (totalMessages / elapsed) * 1000;
      
      console.log(`\n  Clients: ${numClients}`);
      console.log(`  Messages: ${iterations}`);
      console.log(`  Total Deliveries: ${totalMessages}`);
      console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
      console.log(`  Throughput: ${opsPerSec.toFixed(0)} deliveries/sec`);
      
      handlers.forEach(h => clearInterval(h));
      subs.forEach(s => s.close());
      server.stop();
      
      // Client 0 shouldn't receive own messages
      expect(counters[0]).toBe(0);
      // Other clients should receive all messages
      for (let i = 1; i < numClients; i++) {
        expect(counters[i]).toBe(iterations);
      }
    }, 30000);
  });

  describe('Large Payload (1MB)', () => {
    it('should handle large messages', async () => {
      const server = new Server(':0');
      server.start();
      await sleep(50);
      
      const channelID = randomUUID();
      const client1 = new Client(server.addr(), channelID);
      const client2 = new Client(server.addr(), channelID);
      
      const sub1 = client1.subscribe();
      const sub2 = client2.subscribe();
      
      const largeData = Buffer.alloc(1024 * 1024, 'X');
      
      const handler = setInterval(() => {
        const msg = sub2.receive();
        if (msg) client2.publish(msg);
      }, 1);
      
      await sleep(100);
      
      const iterations = 50;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        client1.publish(largeData);
        while (true) {
          const resp = sub1.receive();
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
      sub1.close();
      sub2.close();
      server.stop();
      
      expect(avgLatency).toBeLessThan(200);
    }, 60000);
  });

  describe('Medium Payload (10KB)', () => {
    it('should handle medium-sized messages', async () => {
      const server = new Server(':0');
      server.start();
      await sleep(50);
      
      const channelID = randomUUID();
      const client1 = new Client(server.addr(), channelID);
      const client2 = new Client(server.addr(), channelID);
      
      const sub1 = client1.subscribe();
      const sub2 = client2.subscribe();
      
      const mediumData = Buffer.alloc(10 * 1024, 'X');
      
      const handler = setInterval(() => {
        const msg = sub2.receive();
        if (msg) client2.publish(msg);
      }, 1);
      
      await sleep(100);
      
      const iterations = 1000;
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        client1.publish(mediumData);
        while (true) {
          const resp = sub1.receive();
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
      sub1.close();
      sub2.close();
      server.stop();
      
      expect(avgLatency).toBeLessThan(100);
    }, 30000);
  });
});

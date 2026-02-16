import { Server, Client } from '../registry.mjs';
import { randomUUID } from 'crypto';

describe('Broker Performance Benchmarks', () => {

  describe('Single Request/Response', () => {
    it('should measure latency and throughput', (done) => {
      const server = new Server(':0');
      server.start();

      const channelID = randomUUID();
      const client1 = new Client(server.addr(), channelID);
      const client2 = new Client(server.addr(), channelID);

      client2.onMessage((msg) => {
        client2.publish(msg);
      });

      const iterations = 1000;
      let count = 0;
      const start = process.hrtime.bigint();

      client1.onMessage(() => {
        count++;
        if (count === iterations) {
          const end = process.hrtime.bigint();
          const elapsed = Number(end - start) / 1e6;
          const avgLatency = elapsed / iterations;
          const opsPerSec = (iterations / elapsed) * 1000;

          console.log(`\n  Iterations: ${iterations}`);
          console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
          console.log(`  Avg Latency: ${avgLatency.toFixed(3)} ms`);
          console.log(`  Throughput: ${opsPerSec.toFixed(0)} ops/sec`);

          server.stop();
          expect(avgLatency).toBeLessThan(100);
          done();
        }
      });

      for (let i = 0; i < iterations; i++) {
        client1.publish('test');
      }
    }, 30000);
  });

  describe('Publish Throughput', () => {
    it('should handle high message volume', (done) => {
      const server = new Server(':0');
      server.start();

      const channelID = randomUUID();
      const client1 = new Client(server.addr(), channelID);
      const client2 = new Client(server.addr(), channelID);

      const iterations = 100;
      let received = 0;
      const start = process.hrtime.bigint();

      client2.onMessage(() => {
        received++;
        if (received === iterations) {
          const end = process.hrtime.bigint();
          const elapsed = Number(end - start) / 1e6;
          const opsPerSec = (iterations / elapsed) * 1000;

          console.log(`\n  Iterations: ${iterations}`);
          console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
          console.log(`  Throughput: ${opsPerSec.toFixed(0)} ops/sec`);

          server.stop();
          expect(received).toBe(iterations);
          done();
        }
      });

      for (let i = 0; i < iterations; i++) {
        client1.publish('test');
      }
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
      const counters = [];
      const handlers = [];

      for (let i = 0; i < numClients; i++) {
        const client = new Client(server.addr(), channelID);
        clients.push(client);
        counters.push(0);

        const handler = setInterval(((idx) => () => {
          const msg = clients[idx].subscribe();
          if (msg) counters[idx]++;
        })(i), 0);
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
      server.stop();

      expect(counters[0]).toBe(0);
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

      const largeData = Buffer.alloc(1024 * 1024, 'X');

      const handler = setInterval(() => {
        const msg = client2.subscribe();
        if (msg) client2.publish(msg);
      }, 0);

      await sleep(100);

      const iterations = 50;
      const start = process.hrtime.bigint();

      for (let i = 0; i < iterations; i++) {
        client1.publish(largeData);
        let resp;
        do { resp = client1.subscribe(); } while (!resp);
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

      const mediumData = Buffer.alloc(10 * 1024, 'X');

      const handler = setInterval(() => {
        const msg = client2.subscribe();
        if (msg) client2.publish(msg);
      }, 0);

      await sleep(100);

      const iterations = 1000;
      const start = process.hrtime.bigint();

      for (let i = 0; i < iterations; i++) {
        client1.publish(mediumData);
        let resp;
        do { resp = client1.subscribe(); } while (!resp);
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
      server.stop();

      expect(avgLatency).toBeLessThan(100);
    }, 30000);
  });
});

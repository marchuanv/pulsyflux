import { Server, Client } from './registry.mjs';
import { randomUUID } from 'crypto';

describe('Broker Performance Benchmarks', () => {

  describe('Publish', () => {
    it('should measure publish latency', (done) => {
      const server = new Server(':0');
      server.start();

      setTimeout(() => {
        const channelID = randomUUID();
        const client = new Client(server.addr(), channelID);
        const payload = 'benchmark message';
        const iterations = 100;

        const start = process.hrtime.bigint();
        for (let i = 0; i < iterations; i++) {
          client.publish(payload);
        }
        const end = process.hrtime.bigint();
        
        const elapsed = Number(end - start) / 1e6;
        const avgLatency = elapsed / iterations;
        const opsPerSec = (iterations / elapsed) * 1000;

        console.log(`\n  Iterations: ${iterations}`);
        console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
        console.log(`  Avg Latency: ${avgLatency.toFixed(3)} ms`);
        console.log(`  Throughput: ${opsPerSec.toFixed(0)} ops/sec`);

        server.stop();
        done();
      }, 100);
    }, 3000);
  });

  describe('PubSub', () => {
    it('should measure round-trip latency', (done) => {
      const server = new Server(':0');
      server.start();

      setTimeout(() => {
        const channelID = randomUUID();
        const client1 = new Client(server.addr(), channelID);
        const client2 = new Client(server.addr(), channelID);
        const payload = 'benchmark message';
        const iterations = 10;
        
        let count = 0;
        const start = process.hrtime.bigint();

        client2.onMessage(() => {
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
            done();
          }
        });

        setTimeout(() => {
          for (let i = 0; i < iterations; i++) {
            client1.publish(payload);
          }
        }, 200);
      }, 100);
    }, 3000);
  });

  describe('Broadcast2', () => {
    it('should measure broadcast to 2 clients', (done) => {
      const server = new Server(':0');
      server.start();

      setTimeout(() => {
        const channelID = randomUUID();
        const publisher = new Client(server.addr(), channelID);
        const receiver1 = new Client(server.addr(), channelID);
        const payload = 'benchmark message';
        const iterations = 5;
        
        let count = 0;
        const start = process.hrtime.bigint();

        receiver1.onMessage(() => {
          count++;
          if (count === iterations) {
            const end = process.hrtime.bigint();
            const elapsed = Number(end - start) / 1e6;
            const avgLatency = elapsed / iterations;
            const opsPerSec = (iterations / elapsed) * 1000;

            console.log(`\n  Clients: 2`);
            console.log(`  Iterations: ${iterations}`);
            console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
            console.log(`  Avg Latency: ${avgLatency.toFixed(3)} ms`);
            console.log(`  Throughput: ${opsPerSec.toFixed(0)} ops/sec`);

            server.stop();
            done();
          }
        });

        setTimeout(() => {
          for (let i = 0; i < iterations; i++) {
            publisher.publish(payload);
          }
        }, 200);
      }, 100);
    }, 3000);
  });

  describe('Multiple Channels', () => {
    it('should measure independent channel performance', (done) => {
      const server = new Server(':0');
      server.start();

      setTimeout(() => {
        const ch1 = randomUUID();
        const ch2 = randomUUID();
        const ch3 = randomUUID();

        const c1 = new Client(server.addr(), ch1);
        const c2 = new Client(server.addr(), ch2);
        const c3 = new Client(server.addr(), ch3);

        const payload = 'benchmark message';
        const iterations = 20;

        const start = process.hrtime.bigint();
        for (let i = 0; i < iterations; i++) {
          c1.publish(payload);
          c2.publish(payload);
          c3.publish(payload);
        }
        const end = process.hrtime.bigint();
        
        const totalOps = iterations * 3;
        const elapsed = Number(end - start) / 1e6;
        const avgLatency = elapsed / totalOps;
        const opsPerSec = (totalOps / elapsed) * 1000;

        console.log(`\n  Channels: 3`);
        console.log(`  Iterations per channel: ${iterations}`);
        console.log(`  Total Operations: ${totalOps}`);
        console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
        console.log(`  Avg Latency: ${avgLatency.toFixed(3)} ms`);
        console.log(`  Throughput: ${opsPerSec.toFixed(0)} ops/sec`);

        server.stop();
        done();
      }, 100);
    }, 3000);
  });
});
import { Server, Client } from '../registry.mjs';
import { randomUUID } from 'crypto';

describe('Broker Performance Benchmarks', () => {

  describe('Simple Throughput', () => {
    it('should measure message throughput', (done) => {
      const server = new Server(':0');
      server.start();

      const channelID = randomUUID();
      const publisher = new Client(server.addr(), channelID);
      const receiver = new Client(server.addr(), channelID);

      const iterations = 5;
      let count = 0;
      const start = process.hrtime.bigint();

      receiver.onMessage(() => {
        count++;
        if (count === iterations) {
          const end = process.hrtime.bigint();
          const elapsed = Number(end - start) / 1e6;
          const opsPerSec = (iterations / elapsed) * 1000;

          console.log(`\n  Iterations: ${iterations}`);
          console.log(`  Total Time: ${elapsed.toFixed(2)} ms`);
          console.log(`  Throughput: ${opsPerSec.toFixed(0)} ops/sec`);

          server.stop();
          expect(count).toBe(iterations);
          done();
        }
      });

      // Small delay to ensure receiver is ready
      setTimeout(() => {
        for (let i = 0; i < iterations; i++) {
          publisher.publish(`test${i}`);
        }
      }, 10);
    }, 5000);
  });
});

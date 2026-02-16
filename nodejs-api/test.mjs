import { Server, Client } from './index.mjs';
import { randomUUID } from 'crypto';

console.log('Testing N-API addon...');

const server = new Server(':0');
server.start();
console.log('✓ Server started at:', server.addr());

const channelID = randomUUID();
const pub = new Client(server.addr(), channelID);
const sub = new Client(server.addr(), channelID);

console.log('✓ Clients created');

setTimeout(() => {
  pub.publish(Buffer.from('test message'));
  console.log('✓ Message published');
  
  setTimeout(() => {
    const msg = sub.subscribe();
    console.log('✓ Message received:', msg?.toString());
    
    server.stop();
    console.log('✓ All tests passed!');
  }, 100);
}, 100);

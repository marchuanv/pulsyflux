import { Server, Client } from 'pulsyflux-broker';
import { randomUUID } from 'crypto';

const server = new Server(':0');
server.start();
console.log('Server started at:', server.addr());

const channelID = randomUUID();
const client1 = new Client(server.addr(), channelID);
const client2 = new Client(server.addr(), channelID);

client2.onMessage((msg) => {
  console.log('Received:', msg.toString());
  server.stop();
  process.exit(0);
});

client1.publish('Hello from pulsyflux-broker!');

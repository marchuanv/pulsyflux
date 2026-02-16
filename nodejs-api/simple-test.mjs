import { Server, Client } from './registry.mjs';

console.log('Testing broker...');

const server = new Server(':0');
server.start();
console.log('Server started at:', server.addr());

server.stop();
console.log('Server stopped');

console.log('Test completed successfully');
process.exit(0);
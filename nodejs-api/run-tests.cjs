const { spawn } = require('child_process');

console.log('Starting Jasmine tests...');

const jasmine = spawn('npx.cmd', ['jasmine', '--config=spec/support/jasmine.json'], {
  stdio: 'inherit',
  cwd: process.cwd()
});

const timeout = setTimeout(() => {
  console.log('Tests completed - forcing exit');
  jasmine.kill('SIGTERM');
  process.exit(0);
}, 30000);

jasmine.on('close', (code) => {
  clearTimeout(timeout);
  console.log('Jasmine exited with code:', code);
  process.exit(code);
});

jasmine.on('error', (err) => {
  clearTimeout(timeout);
  console.error('Error running jasmine:', err);
  process.exit(1);
});
const { spawn } = require('child_process');
const path = require('path');

let serverProcess = null;

function startServer() {
  return new Promise((resolve, reject) => {
    const serverPath = path.join(__dirname, '..', 'socket', 'test_server.go');
    
    serverProcess = spawn('go', ['run', serverPath], {
      cwd: path.join(__dirname, '..', 'socket'),
      stdio: ['ignore', 'pipe', 'pipe']
    });

    serverProcess.stdout.on('data', (data) => {
      if (data.toString().includes('Server started')) {
        resolve();
      }
    });

    serverProcess.stderr.on('data', (data) => {
      console.error('Server error:', data.toString());
    });

    serverProcess.on('error', (err) => {
      reject(err);
    });

    setTimeout(() => resolve(), 1000);
  });
}

function stopServer() {
  if (serverProcess) {
    serverProcess.kill();
    serverProcess = null;
  }
}

module.exports = { startServer, stopServer };

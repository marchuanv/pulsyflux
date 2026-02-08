import { Worker, isMainThread, parentPort, workerData } from 'worker_threads';
import crypto from 'crypto';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);

/**@type { Map<crypto.UUID, Promise<string>> } */
const providerQueue = new Map();

/**@type { Map<crypto.UUID, Promise<string>> } */
const consumerQueue = new Map();

class Consumer {
  constructor(address, channelID) {
    this.worker = new Worker(__filename, {
      workerData: { type: 'consumer', address, channelID }
    });
  }

  async send(data, timeoutMs = 5000) {
    return new Promise((resolve, reject) => {
      this.worker.once('message', (result) => {
        if (result.error) {
          reject(new Error(result.error));
        } else {
          resolve(result.data)
        };
      });
      this.worker.postMessage({
        action: 'send',
        data, timeoutMs
      });
    });
  }

  async close() {
    if (this.worker) {
      try {
        // Wait for worker to acknowledge close
        const closePromise = new Promise((resolve) => {
          this.worker.once('message', (result) => {
            if (result.action === 'closed') {
              resolve();
            }
          });
        });

        this.worker.postMessage({ action: 'close' });
        await closePromise;
        await this.worker.terminate();
      } catch (error) {
        // Force terminate if cleanup fails
        try {
          await this.worker.terminate();
        } catch (e) { }
      }
      this.worker = null;
    }
  }
}

class Provider {
  constructor(address, channelID) {
    this.worker = new Worker(__filename, {
      workerData: { type: 'provider', address, channelID }
    });
    this.requestQueue = [];
    this.pendingResolves = [];

    this.worker.on('message', (result) => {
      if (result.type === 'request') {
        if (this.pendingResolves.length > 0) {
          const resolve = this.pendingResolves.shift();
          resolve(result.data);
        } else {
          this.requestQueue.push(result.data);
        }
      }
    });

    // Start polling immediately
    this.worker.postMessage({ action: 'startPolling' });
  }

  async receive() {
    if (this.requestQueue.length > 0) {
      return this.requestQueue.shift();
    }

    return new Promise((resolve) => {
      this.pendingResolves.push(resolve);
    });
  }

  async respond(requestID, data, error = null) {
    this.worker.postMessage({ action: 'respond', requestID, data, error });
  }

  async close() {
    if (this.worker) {
      try {
        // Wait for worker to acknowledge close
        const closePromise = new Promise((resolve) => {
          this.worker.once('message', (result) => {
            if (result.action === 'closed') {
              resolve();
            }
          });
        });
        this.worker.postMessage({ action: 'close' });
        await closePromise;
        await this.worker.terminate();
      } catch (error) {
        // Force terminate if cleanup fails
        try {
          await this.worker.terminate();
        } catch (e) { }
      }
      this.worker = null;
    }
    // Resolve any pending receives with null
    this.pendingResolves.forEach(resolve => resolve(null));
    this.pendingResolves = [];
  }
}

class Server {
  constructor(port) {
    this.worker = new Worker(__filename, {
      workerData: { type: 'server', port }
    });
  }

  async start() {
    return new Promise((resolve, reject) => {
      this.worker.once('message', (result) => {
        if (result.error) {
          reject(new Error(result.error));
        } else {
          resolve();
        }
      });
      this.worker.postMessage({ action: 'start' });
    });
  }

  async stop() {
    if (this.worker) {
      try {
        // Wait for worker to acknowledge stop
        const stopPromise = new Promise((resolve) => {
          this.worker.once('message', (result) => {
            if (result.action === 'stopped') {
              resolve();
            }
          });
        });

        this.worker.postMessage({ action: 'stop' });
        await stopPromise;
        await this.worker.terminate();
      } catch (error) {
        // Force terminate if cleanup fails
        try {
          await this.worker.terminate();
        } catch (e) { }
      }
      this.worker = null;
    }
  }
}

if (!isMainThread) {
  const { type, address, channelID, port } = workerData;

  (async () => {
    const { Consumer: RawConsumer, Provider: RawProvider, Server: RawServer } = await import('./socket.mjs');
    let instance;


    if (type === 'consumer') {
      instance = new RawConsumer(address, channelID);
    } else if (type === 'provider') {
      instance = new RawProvider(address, channelID);
    } else if (type === 'server') {
      instance = new RawServer(port);
    } else {
      throw new Error(`Unknown type: ${type}`);
    }

    parentPort.on('message', (msg) => {
      try {
        if (msg.action === 'send') {
          const result = instance.send(msg.data, msg.timeoutMs);
          parentPort.postMessage({ data: result });
        } else if (msg.action === 'start') {
          const error = instance.start();
          parentPort.postMessage({ error });
        } else if (msg.action === 'stop') {
          instance.stop();
          parentPort.postMessage({ action: 'stopped' });
        } else if (msg.action === 'close') {
          instance.close();
          parentPort.postMessage({ action: 'closed' });
        } else if (msg.action === 'startPolling') {
          polling = true;
          parentPort.postMessage({ type: 'request', data: req });
        } else if (msg.action === 'stopPolling') {
          polling = false;
        } else if (msg.action === 'respond') {
          instance.respond(msg.requestID, msg.data, msg.error);
        }
      } catch (error) {
        parentPort.postMessage({ error: error.message });
      }
    });
  })();
}

export { Consumer, Provider, Server };

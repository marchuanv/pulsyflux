import { Worker, isMainThread, parentPort, workerData } from 'worker_threads';
import { Consumer as RawConsumer, Provider as RawProvider, Server as RawServer } from './socket.mjs';

class Consumer {
  constructor(address, channelID) {
    this.worker = new Worker(new URL(import.meta.url), {
      workerData: { type: 'consumer', address, channelID }
    });
    this.ready = new Promise(resolve => {
      this.worker.once('online', resolve);
    });
  }

  async send(data, timeoutMs = 5000) {
    await this.ready;
    return new Promise((resolve, reject) => {
      this.worker.postMessage({ action: 'send', data, timeoutMs });
      this.worker.once('message', (result) => {
        if (result.error) reject(new Error(result.error));
        else resolve(result.data);
      });
    });
  }

  async close() {
    await this.ready;
    this.worker.terminate();
  }
}

class Provider {
  constructor(address, channelID) {
    this.worker = new Worker(new URL(import.meta.url), {
      workerData: { type: 'provider', address, channelID }
    });
    this.ready = new Promise(resolve => {
      this.worker.once('online', resolve);
    });
  }

  async receive() {
    await this.ready;
    return new Promise((resolve) => {
      this.worker.postMessage({ action: 'receive' });
      this.worker.once('message', (result) => {
        resolve(result.data);
      });
    });
  }

  async respond(requestID, data, error = null) {
    await this.ready;
    this.worker.postMessage({ action: 'respond', requestID, data, error });
  }

  async close() {
    await this.ready;
    this.worker.terminate();
  }
}

class Server {
  constructor(port) {
    this.worker = new Worker(new URL(import.meta.url), {
      workerData: { type: 'server', port }
    });
    this.ready = new Promise(resolve => {
      this.worker.once('online', resolve);
    });
  }

  async start() {
    await this.ready;
    this.worker.postMessage({ action: 'start' });
  }

  async stop() {
    await this.ready;
    this.worker.terminate();
  }
}

// Worker thread code
if (!isMainThread) {
  const { type, address, channelID, port } = workerData;
  let instance;

  if (type === 'consumer') {
    instance = new RawConsumer(address, channelID);
  } else if (type === 'provider') {
    instance = new RawProvider(address, channelID);
  } else if (type === 'server') {
    instance = new RawServer(port);
  }

  parentPort.on('message', (msg) => {
    try {
      if (msg.action === 'send') {
        const result = instance.send(msg.data, msg.timeoutMs);
        parentPort.postMessage({ data: result });
      } else if (msg.action === 'receive') {
        const result = instance.receive();
        parentPort.postMessage({ data: result });
      } else if (msg.action === 'respond') {
        instance.respond(msg.requestID, msg.data, msg.error);
      } else if (msg.action === 'start') {
        instance.start();
      }
    } catch (error) {
      parentPort.postMessage({ error: error.message });
    }
  });
}

export { Consumer, Provider, Server };
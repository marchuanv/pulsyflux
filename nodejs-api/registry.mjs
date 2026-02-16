import { createRequire } from 'module';
const require = createRequire(import.meta.url);
const { Server, Client: NativeClient } = require('./broker_addon.node');

class Client {
  constructor(addr, channelID) {
    this._client = new NativeClient(addr, channelID);
  }

  publish(data) {
    return this._client.publish(data);
  }

  subscribe() {
    return this._client.subscribe();
  }

  onMessage(callback) {
    this._client.onMessage(callback);
  }

  close() {
    // No-op since Go broker doesn't provide client cleanup
  }
}

export { Server, Client };

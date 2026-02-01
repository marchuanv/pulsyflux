import ffi from 'ffi-napi';
import ref from 'ref-napi';
import { EventEmitter } from 'events';

const stringPtr = ref.refType('string');
const intPtr = ref.refType('int');

const socketLib = ffi.Library('./build/socket_lib', {
  'ConsumerNew': ['int', ['string', 'string']],
  'ConsumerSend': ['string', ['int', 'string', 'int', 'int']],
  'ConsumerClose': ['int', ['int']],
  'ProviderNew': ['int', ['string', 'string']],
  'ProviderReceive': ['int', ['int', stringPtr, stringPtr, intPtr]],
  'ProviderRespond': ['int', ['int', 'string', 'string', 'int', 'string']],
  'ProviderClose': ['int', ['int']],
  'ServerNew': ['int', ['string']],
  'ServerStart': ['int', ['int']],
  'ServerStop': ['int', ['int']],
  'FreeString': ['void', ['string']]
});

class Consumer {
  constructor(address, channelID) {
    this.id = socketLib.ConsumerNew(address, channelID);
    if (this.id < 0) {
      throw new Error('Failed to create consumer');
    }
  }

  send(data, timeoutMs = 5000) {
    const buffer = Buffer.isBuffer(data) ? data : Buffer.from(data);
    const result = socketLib.ConsumerSend(this.id, buffer, buffer.length, timeoutMs);
    if (result === null) {
      throw new Error('Send failed');
    }
    return result;
  }

  close() {
    if (this.id >= 0) {
      socketLib.ConsumerClose(this.id);
      this.id = -1;
    }
  }
}

class Provider extends EventEmitter {
  constructor(address, channelID) {
    super();
    this.id = socketLib.ProviderNew(address, channelID);
    if (this.id < 0) {
      throw new Error('Failed to create provider');
    }
  }

  receive() {
    const reqID = ref.alloc(stringPtr);
    const data = ref.alloc(stringPtr);
    const dataLen = ref.alloc('int');

    try {
      const result = socketLib.ProviderReceive(this.id, reqID, data, dataLen);
      if (result !== 0) {  // 0 = success, -1 = error, -2 = no request
        return null;
      }

      const reqIDStr = reqID.deref();
      const dataPtr = data.deref();
      const dataLenVal = dataLen.deref();

      if (!reqIDStr || !dataPtr) {
        return null;
      }

      return {
        requestID: reqIDStr,
        data: ref.readCString(dataPtr, 0, dataLenVal)
      };
    } catch (error) {
      console.error('ProviderReceive error:', error);
      return null;
    }
  }

  respond(requestID, data, error = null) {
    const buffer = data ? (Buffer.isBuffer(data) ? data : Buffer.from(data)) : null;
    const result = socketLib.ProviderRespond(
      this.id,
      requestID,
      buffer,
      buffer ? buffer.length : 0,
      error
    );
    return result === 0;
  }

  close() {
    if (this.id >= 0) {
      socketLib.ProviderClose(this.id);
      this.id = -1;
    }
  }
}

class Server {
  constructor(port) {
    this.id = socketLib.ServerNew(port);
    if (this.id < 0) {
      throw new Error('Failed to create server');
    }
  }

  start() {
    if (this.id >= 0) {
      const result = socketLib.ServerStart(this.id);
      if (result < 0) {
        throw new Error('Failed to start server');
      }
    }
  }

  stop() {
    if (this.id >= 0) {
      socketLib.ServerStop(this.id);
      this.id = -1;
    }
  }
}

export { Consumer, Provider, Server };

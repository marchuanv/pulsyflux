import ffi from 'ffi-napi';
import ref from 'ref-napi';
import crypto from 'node:crypto';

const charPtr = ref.refType('char');
const stringPtr = ref.refType(charPtr);
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
    const str = result.toString();
    if (str.includes('\n') || str.includes('\r')) {
      throw new Error(str.trim());
    }
    return str;
  }

  close() {
    if (this.id >= 0) {
      socketLib.ConsumerClose(this.id);
      this.id = -1;
    }
  }
}

class Provider {
  #queue;
  #id;
  /**
  * @returns { { requestID: crypto.UUID|null, data: String|null, error: Error|null } }
  */
  #receive() {
    const reqID = ref.alloc(stringPtr);
    const data = ref.alloc(stringPtr);
    const dataLen = ref.alloc('int');

    try {
      const result = socketLib.ProviderReceive(this.#id, reqID, data, dataLen);
      if (result !== 0) {  // 0 = success, -1 = error, -2 = no request
        throw new Error('error processing received data from the provider')
      }

      let reqIDStr = reqID.deref();
      if (!reqIDStr || !dataPtr) {
        throw new Error('error processing received data from the provider')
      }

      reqIDStr = ref.readCString(reqIDStr, 0);

      const dataPtr = data.deref();
      const dataLenVal = dataLen.deref();
      // Use Buffer.from to read exact bytes instead of C string
      const dataBuffer = Buffer.from(ref.reinterpret(dataPtr, dataLenVal, 0));
      return {
        requestID: reqIDStr,
        data: dataBuffer.toString('utf8'),
        error: null
      };
    } catch (error) {
      if (error instanceof Error) {
        return {
          requestID: null,
          data: null,
          error: new Error('error processing received data from the provider')
        };
      } else {
        return {
          requestID: null,
          data: null,
          error: new Error(error)
        };
      }
    }
  }

  constructor(address, channelID) {
    this.#id = socketLib.ProviderNew(address, channelID);
    this.#queue = [];
    if (this.#id < 0) {
      throw new Error('Failed to create provider');
    }
    while (true) {
      const { requestID, data, error } = this.#receive();
      this.#queue.push({ requestID, data, error });
    }
  }

  respond(requestID, data, error = null) {
    const buffer = data ? (Buffer.isBuffer(data) ? data : Buffer.from(data)) : null;
    const result = socketLib.ProviderRespond(
      this.#id,
      requestID,
      buffer,
      buffer ? buffer.length : 0,
      error
    );
    return result === 0;
  }

  close() {
    if (this.#id >= 0) {
      socketLib.ProviderClose(this.#id);
      this.#id = -1;
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

  /**
  * @returns { undefined|Error }
  */
  start() {
    if (this.id >= 0) {
      const result = socketLib.ServerStart(this.id);
      if (result < 0) {
        return new Error('Failed to start server');
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

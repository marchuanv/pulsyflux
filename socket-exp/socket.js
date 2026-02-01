const ffi = require('ffi-napi');
const ref = require('ref-napi');

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
    const buffer = Buffer.from(data);
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

class Provider {
  constructor(address, channelID) {
    this.id = socketLib.ProviderNew(address, channelID);
    if (this.id < 0) {
      throw new Error('Failed to create provider');
    }
  }

  receive() {
    const reqID = ref.alloc(stringPtr);
    const data = ref.alloc(stringPtr);
    const dataLen = ref.alloc('int');

    const result = socketLib.ProviderReceive(this.id, reqID, data, dataLen);
    if (result < 0) {
      return null;
    }

    return {
      requestID: reqID.deref(),
      data: data.deref()
    };
  }

  respond(requestID, data, error = null) {
    const buffer = data ? Buffer.from(data) : null;
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

module.exports = { Consumer, Provider };

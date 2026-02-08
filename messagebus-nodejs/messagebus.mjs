import ffi from 'ffi-napi';
import ref from 'ref-napi';

const stringPtr = ref.refType('string');
const intPtr = ref.refType('int');
const longlongPtr = ref.refType('longlong');

const lib = ffi.Library('./build/messagebus_lib', {
  'BusNew': ['int', ['string', 'string']],
  'BusPublish': ['int', ['int', 'string', 'string', 'int', 'string', 'int']],
  'BusSubscribe': ['int', ['int', 'string']],
  'BusReceive': ['int', ['int', stringPtr, stringPtr, stringPtr, intPtr, stringPtr, longlongPtr]],
  'BusUnsubscribe': ['int', ['int', 'string']],
  'BusClose': ['int', ['int']],
  'SubscriptionClose': ['int', ['int']],
  'ServerNew': ['int', ['string']],
  'ServerStart': ['int', ['int']],
  'ServerStop': ['int', ['int']],
  'FreeString': ['void', ['string']]
});

export class Server {
  constructor(port) {
    this.id = lib.ServerNew(port);
    if (this.id < 0) {
      throw new Error('Failed to create server');
    }
  }

  start() {
    const result = lib.ServerStart(this.id);
    if (result < 0) {
      throw new Error('Failed to start server');
    }
  }

  stop() {
    if (this.id >= 0) {
      lib.ServerStop(this.id);
      this.id = -1;
    }
  }
}

export class Bus {
  constructor(address, channelID) {
    this.id = lib.BusNew(address, channelID);
    if (this.id < 0) {
      throw new Error('Failed to create bus');
    }
  }

  publish(topic, payload, headers = null, timeoutMs = 5000) {
    const payloadBuffer = Buffer.from(payload);
    const headersStr = headers ? JSON.stringify(headers) : null;
    
    const result = lib.BusPublish(
      this.id,
      topic,
      payloadBuffer,
      payloadBuffer.length,
      headersStr,
      timeoutMs
    );

    if (result < 0) {
      throw new Error('Failed to publish message');
    }
  }

  subscribe(topic) {
    const subID = lib.BusSubscribe(this.id, topic);
    if (subID < 0) {
      throw new Error('Failed to subscribe to topic');
    }
    return new Subscription(subID);
  }

  unsubscribe(topic) {
    const result = lib.BusUnsubscribe(this.id, topic);
    if (result < 0) {
      throw new Error('Failed to unsubscribe from topic');
    }
  }

  close() {
    if (this.id >= 0) {
      lib.BusClose(this.id);
      this.id = -1;
    }
  }
}

export class Subscription {
  constructor(id) {
    this.id = id;
  }

  receive() {
    const msgID = ref.alloc(stringPtr);
    const topic = ref.alloc(stringPtr);
    const payload = ref.alloc(stringPtr);
    const payloadLen = ref.alloc('int');
    const headers = ref.alloc(stringPtr);
    const timestamp = ref.alloc('longlong');

    const result = lib.BusReceive(
      this.id,
      msgID,
      topic,
      payload,
      payloadLen,
      headers,
      timestamp
    );

    if (result === -3) {
      return null; // No message available
    }

    if (result < 0) {
      return null; // Channel closed or error
    }

    const msg = {
      id: msgID.deref(),
      topic: topic.deref(),
      payload: Buffer.from(payload.deref(), payloadLen.deref()),
      timestamp: new Date(Number(timestamp.deref()) * 1000),
      headers: null
    };

    const headersStr = headers.deref();
    if (headersStr) {
      msg.headers = JSON.parse(headersStr);
    }

    return msg;
  }

  close() {
    if (this.id >= 0) {
      lib.SubscriptionClose(this.id);
      this.id = -1;
    }
  }

  async *[Symbol.asyncIterator]() {
    while (true) {
      const msg = this.receive();
      if (msg === null) {
        await new Promise(resolve => setTimeout(resolve, 10));
        continue;
      }
      yield msg;
    }
  }
}

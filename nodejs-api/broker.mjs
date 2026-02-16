import ffi from 'ffi-napi';
import ref from 'ref-napi';

const stringPtr = ref.refType('string');
const intPtr = ref.refType('int');
const voidPtr = ref.refType(ref.types.void);
const voidPtrPtr = ref.refType(voidPtr);

const lib = ffi.Library('./broker_lib', {
  'ClientNew': ['int', ['string', 'string']],
  'ClientPublish': ['int', ['int', 'string', 'int']],
  'ClientSubscribe': ['int', ['int']],
  'SubscriptionReceive': ['int', ['int', voidPtrPtr, intPtr]],
  'SubscriptionClose': ['int', ['int']],
  'ServerNew': ['int', ['string']],
  'ServerStart': ['int', ['int']],
  'ServerAddr': ['string', ['int']],
  'ServerStop': ['int', ['int']],
  'FreeString': ['void', ['string']]
});

export class Server {
  constructor(address) {
    this.id = lib.ServerNew(address);
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

  addr() {
    return lib.ServerAddr(this.id);
  }

  stop() {
    if (this.id >= 0) {
      lib.ServerStop(this.id);
      this.id = -1;
    }
  }
}

export class Client {
  constructor(address, channelID) {
    this.id = lib.ClientNew(address, channelID);
    if (this.id < 0) {
      throw new Error('Failed to create client');
    }
  }

  publish(payload) {
    const payloadBuffer = Buffer.from(payload);
    
    const result = lib.ClientPublish(
      this.id,
      payloadBuffer,
      payloadBuffer.length
    );

    if (result < 0) {
      throw new Error('Failed to publish message');
    }
  }

  subscribe() {
    const subID = lib.ClientSubscribe(this.id);
    if (subID < 0) {
      throw new Error('Failed to subscribe');
    }
    return new Subscription(subID);
  }
}

export class Subscription {
  constructor(id) {
    this.id = id;
  }

  receive() {
    const payload = ref.alloc(voidPtrPtr);
    const payloadLen = ref.alloc('int');

    const result = lib.SubscriptionReceive(
      this.id,
      payload,
      payloadLen
    );

    if (result === -3) {
      return null; // No message available
    }

    if (result < 0) {
      return null; // Channel closed or error
    }

    const len = payloadLen.deref();
    const payloadPtr = payload.deref();
    const payloadBuf = ref.reinterpret(payloadPtr, len, 0);

    return payloadBuf;
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

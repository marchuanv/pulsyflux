import { parentPort } from 'node:worker_threads';
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

const consumers = new Map();

parentPort.onmessage(({ address, channel, data, close = false }) => {

    let consumer = consumers.get(address);
    if (!consumer) {
        consumers.set(address, consumer);
    }

    const consumer = new Consumer(address, channel);
    consumer.send(data, 5000);
});
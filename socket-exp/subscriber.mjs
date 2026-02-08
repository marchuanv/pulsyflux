import { Worker } from 'node:worker_threads';
import { URL } from 'node:url';

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

export class Subscriber {
    constructor() {
        const worker = new Worker(new URL('./subscriber-worker.mjs', import.meta.url));
    }
}


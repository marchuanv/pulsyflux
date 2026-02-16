export declare class Server {
  constructor(address: string);
  start(): void;
  addr(): string;
  stop(): void;
}

export declare class Client {
  constructor(address: string, channelID: string);
  publish(payload: string | Buffer): void;
  subscribe(): Buffer | null;
  onMessage(callback: (message: Buffer) => void): void;
  close(): void;
}
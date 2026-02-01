const { Consumer, Provider, Server } = require('./socket');

describe('Simple Socket Tests', () => {
  it('should create objects without server', () => {
    // Test object creation only - no network calls
    expect(() => new Consumer('127.0.0.1:9999', 'test-channel')).toThrow();
    expect(() => new Provider('127.0.0.1:9999', 'test-channel')).toThrow();
    
    const server = new Server('9091');
    expect(server.id).toBeGreaterThanOrEqual(0);
    server.stop();
  });
});
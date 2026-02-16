package tcpconn

import (
	"context"
	"github.com/google/uuid"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultIdleTimeout = 30 * time.Second
	chunkSize          = 64 * 1024 // 64KB chunks
)

type state int

const (
	stateConnecting state = iota
	stateConnected
	stateDisconnected
)

type Connection struct {
	id          uuid.UUID
	address     string
	conn        net.Conn
	state       state
	mu          sync.RWMutex
	readBuf     []byte
	lastRead    time.Time
	lastWrite   time.Time
	idleTimeout time.Duration
	closed      int32
	readInUse   int32
	writeInUse  int32
	ctx         context.Context
	cancel      context.CancelFunc
	recvChan    chan []byte
	wrapped     bool
	wrappedPool *wrappedPool
}

func NewConnection(address string, id uuid.UUID) *Connection {
	ctx, cancel := context.WithCancel(context.Background())

	_, err := globalPool.getOrCreate(address)
	if err != nil {
		cancel()
		return nil
	}

	recvChan := make(chan []byte, 10)
	globalPool.register(address, id, recvChan)

	tc := &Connection{
		id:          id,
		address:     address,
		state:       stateConnected,
		readBuf:     make([]byte, 4096),
		lastRead:    time.Now(),
		lastWrite:   time.Now(),
		idleTimeout: defaultIdleTimeout,
		ctx:         ctx,
		cancel:      cancel,
		recvChan:    recvChan,
		wrapped:     false,
	}

	go tc.idleMonitor()

	return tc
}

func WrapConnection(conn net.Conn, id uuid.UUID) *Connection {
	ctx, cancel := context.WithCancel(context.Background())

	pool := getOrCreateWrappedPool(conn)
	recvChan := make(chan []byte, 10)
	pool.register(id, recvChan)

	tc := &Connection{
		id:          id,
		conn:        conn,
		state:       stateConnected,
		readBuf:     make([]byte, 4096),
		lastRead:    time.Now(),
		lastWrite:   time.Now(),
		idleTimeout: defaultIdleTimeout,
		ctx:         ctx,
		cancel:      cancel,
		recvChan:    recvChan,
		wrapped:     true,
		wrappedPool: pool,
	}

	go tc.idleMonitor()

	return tc
}

func (t *Connection) idleMonitor() {
	ticker := time.NewTicker(t.idleTimeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			t.mu.RLock()
			lastActivity := t.lastRead
			if t.lastWrite.After(t.lastRead) {
				lastActivity = t.lastWrite
			}
			idle := time.Since(lastActivity)
			t.mu.RUnlock()

			if idle > t.idleTimeout {
				t.close()
				return
			}
		}
	}
}

func (t *Connection) reconnect() error {
	_, err := globalPool.getOrCreate(t.address)
	return err
}

func (t *Connection) ensureConnected() error {
	if t.state == stateDisconnected {
		if t.address != "" {
			if err := t.reconnect(); err != nil {
				return err
			}
			t.state = stateConnected
			go t.idleMonitor()
		} else {
			return errConnectionClosed
		}
	}
	return nil
}

func (t *Connection) readFull(buf []byte) error {
	var conn net.Conn
	if t.conn != nil {
		conn = t.conn
	} else {
		conn = globalPool.get(t.address)
		if conn == nil {
			t.state = stateDisconnected
			return errConnectionClosed
		}
	}

	offset := 0
	for offset < len(buf) {
		n, err := conn.Read(buf[offset:])
		if err != nil {
			t.state = stateDisconnected
			return err
		}
		offset += n
	}
	return nil
}

func (t *Connection) Send(data []byte) error {
	if !atomic.CompareAndSwapInt32(&t.writeInUse, 0, 1) {
		return errConnectionInUse
	}
	defer atomic.StoreInt32(&t.writeInUse, 0)

	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.ensureConnected(); err != nil {
		return err
	}

	var conn net.Conn
	if t.conn != nil {
		conn = t.conn
	} else {
		conn = globalPool.get(t.address)
		if conn == nil {
			t.state = stateDisconnected
			return errConnectionClosed
		}
	}

	idBytes := []byte(t.id.String())
	idLen := byte(len(idBytes))
	totalLen := uint32(len(data))

	for offset := 0; offset < len(data); offset += chunkSize {
		end := offset + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[offset:end]
		chunkLen := uint32(len(chunk))

		frame := make([]byte, 1+len(idBytes)+4+4+len(chunk))
		frame[0] = idLen
		copy(frame[1:], idBytes)
		pos := 1 + len(idBytes)
		frame[pos] = byte(totalLen >> 24)
		frame[pos+1] = byte(totalLen >> 16)
		frame[pos+2] = byte(totalLen >> 8)
		frame[pos+3] = byte(totalLen)
		pos += 4
		frame[pos] = byte(chunkLen >> 24)
		frame[pos+1] = byte(chunkLen >> 16)
		frame[pos+2] = byte(chunkLen >> 8)
		frame[pos+3] = byte(chunkLen)
		pos += 4
		copy(frame[pos:], chunk)

		frameOffset := 0
		for frameOffset < len(frame) {
			n, err := conn.Write(frame[frameOffset:])
			if err != nil {
				t.state = stateDisconnected
				return err
			}
			frameOffset += n
		}
	}

	t.lastWrite = time.Now()
	return nil
}

func (t *Connection) Receive() ([]byte, error) {
	if !atomic.CompareAndSwapInt32(&t.readInUse, 0, 1) {
		return nil, errConnectionInUse
	}
	defer atomic.StoreInt32(&t.readInUse, 0)

	t.mu.Lock()
	if err := t.ensureConnected(); err != nil {
		t.mu.Unlock()
		return nil, err
	}
	t.mu.Unlock()

	select {
	case data, ok := <-t.recvChan:
		if !ok {
			return nil, errConnectionClosed
		}
		t.mu.Lock()
		t.lastRead = time.Now()
		t.mu.Unlock()
		return data, nil
	case <-t.ctx.Done():
		return nil, errConnectionClosed
	}
}

func (t *Connection) close() error {
	if !atomic.CompareAndSwapInt32(&t.closed, 0, 1) {
		return nil
	}

	t.mu.Lock()
	t.state = stateDisconnected
	t.mu.Unlock()

	t.cancel()
	if t.address != "" {
		globalPool.unregister(t.address, t.id)
		globalPool.release(t.address)
	} else if t.wrappedPool != nil {
		t.wrappedPool.unregister(t.id)
	}
	return nil
}

var wrappedPools = struct {
	pools map[net.Conn]*wrappedPool
	mu    sync.Mutex
}{
	pools: make(map[net.Conn]*wrappedPool),
}

type wrappedPool struct {
	conn    net.Conn
	demuxer *demuxer
	ctx     context.Context
	cancel  context.CancelFunc
}

func getOrCreateWrappedPool(conn net.Conn) *wrappedPool {
	wrappedPools.mu.Lock()
	defer wrappedPools.mu.Unlock()

	if pool, exists := wrappedPools.pools[conn]; exists {
		return pool
	}

	ctx, cancel := context.WithCancel(context.Background())
	pool := &wrappedPool{
		conn: conn,
		demuxer: &demuxer{
			conn:   conn,
			routes: make(map[uuid.UUID]chan []byte),
			ctx:    ctx,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	wrappedPools.pools[conn] = pool
	go pool.demuxer.run()

	return pool
}

func (wp *wrappedPool) register(id uuid.UUID, ch chan []byte) {
	wp.demuxer.register(id, ch)
}

func (wp *wrappedPool) unregister(id uuid.UUID) {
	wp.demuxer.unregister(id)
}

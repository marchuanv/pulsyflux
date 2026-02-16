package tcpconn

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultIdleTimeout = 5 * time.Minute
)

type state int

const (
	stateConnecting state = iota
	stateConnected
	stateDisconnected
)

type Connection struct {
	id          string
	address     string
	connectFn   func() (net.Conn, error)
	conn        net.Conn // For wrapped connections only
	state       state
	mu          sync.RWMutex
	readBuf     []byte
	lastRead    time.Time
	lastWrite   time.Time
	idleTimeout time.Duration
	closed      int32
	ready       int32
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewConnection(address string, idleTimeout time.Duration) *Connection {
	return NewConnectionWithID(address, "", idleTimeout)
}

func NewConnectionWithID(address string, id string, idleTimeout time.Duration) *Connection {
	if idleTimeout == 0 {
		idleTimeout = defaultIdleTimeout
	}

	ctx, cancel := context.WithCancel(context.Background())

	connectFn := func() (net.Conn, error) {
		return net.Dial("tcp", address)
	}

	_, err := globalPool.getOrCreate(address)
	if err != nil {
		cancel()
		return nil
	}

	tc := &Connection{
		id:          id,
		address:     address,
		connectFn:   connectFn,
		state:       stateConnected,
		readBuf:     make([]byte, 4096),
		lastRead:    time.Now(),
		lastWrite:   time.Now(),
		idleTimeout: idleTimeout,
		ctx:         ctx,
		cancel:      cancel,
	}

	go tc.idleMonitor()

	if id != "" {
		if err := tc.waitReady(); err != nil {
			tc.close()
			return nil
		}
	}

	return tc
}

func WrapConnection(conn net.Conn, idleTimeout time.Duration) *Connection {
	return WrapConnectionWithID(conn, "", idleTimeout)
}

func WrapConnectionWithID(conn net.Conn, id string, idleTimeout time.Duration) *Connection {
	if idleTimeout == 0 {
		idleTimeout = defaultIdleTimeout
	}

	ctx, cancel := context.WithCancel(context.Background())

	tc := &Connection{
		id:          id,
		conn:        conn,
		state:       stateConnected,
		readBuf:     make([]byte, 4096),
		lastRead:    time.Now(),
		lastWrite:   time.Now(),
		idleTimeout: idleTimeout,
		ctx:         ctx,
		cancel:      cancel,
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
		if t.connectFn != nil {
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

	// Frame: [id_len(1)][id][data_len(4)][data]
	idBytes := []byte(t.id)
	idLen := byte(len(idBytes))
	dataLen := uint32(len(data))

	frame := make([]byte, 1+len(idBytes)+4+len(data))
	frame[0] = idLen
	copy(frame[1:], idBytes)
	frame[1+len(idBytes)] = byte(dataLen >> 24)
	frame[2+len(idBytes)] = byte(dataLen >> 16)
	frame[3+len(idBytes)] = byte(dataLen >> 8)
	frame[4+len(idBytes)] = byte(dataLen)
	copy(frame[5+len(idBytes):], data)

	offset := 0
	for offset < len(frame) {
		n, err := conn.Write(frame[offset:])
		if err != nil {
			t.state = stateDisconnected
			return err
		}
		offset += n
	}

	t.lastWrite = time.Now()
	return nil
}

func (t *Connection) waitReady() error {
	if atomic.LoadInt32(&t.ready) == 1 {
		return nil
	}

	handshake := []byte("HANDSHAKE:" + t.id)
	if err := t.Send(handshake); err != nil {
		return err
	}

	if _, err := t.Receive(); err != nil {
		return err
	}

	atomic.StoreInt32(&t.ready, 1)
	return nil
}

func (t *Connection) Receive() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.ensureConnected(); err != nil {
		return nil, err
	}

	for {
		// Read frame: [id_len(1)][id][data_len(4)][data]
		idLenBuf := make([]byte, 1)
		if err := t.readFull(idLenBuf); err != nil {
			return nil, err
		}

		idLen := int(idLenBuf[0])
		idBuf := make([]byte, idLen)
		if err := t.readFull(idBuf); err != nil {
			return nil, err
		}

		dataLenBuf := make([]byte, 4)
		if err := t.readFull(dataLenBuf); err != nil {
			return nil, err
		}

		dataLen := uint32(dataLenBuf[0])<<24 | uint32(dataLenBuf[1])<<16 | uint32(dataLenBuf[2])<<8 | uint32(dataLenBuf[3])
		data := make([]byte, dataLen)
		if err := t.readFull(data); err != nil {
			return nil, err
		}

		receivedID := string(idBuf)
		if t.id != "" && receivedID != t.id {
			// Wrong connection ID, skip this message
			continue
		}

		t.lastRead = time.Now()
		return data, nil
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
		globalPool.release(t.address)
	}
	return nil
}

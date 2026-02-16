package tcpconn

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	errConnectionClosed = errors.New("connection closed")
	errConnectionDead   = errors.New("connection dead")
)

type state int

const (
	stateConnecting state = iota
	stateConnected
	stateDisconnected
)

const (
	defaultIdleTimeout = 5 * time.Minute
)

type Connection struct {
	address     string
	connectFn   func() (net.Conn, error)
	conn        net.Conn
	state       state
	mu          sync.RWMutex
	readBuf     []byte
	lastRead    time.Time
	lastWrite   time.Time
	idleTimeout time.Duration
	closed      int32
	ctx         context.Context
	cancel      context.CancelFunc
	pool        []*net.Conn
	poolSize    int
}

func NewConnection(address string, idleTimeout time.Duration) *Connection {
	if idleTimeout == 0 {
		idleTimeout = defaultIdleTimeout
	}

	ctx, cancel := context.WithCancel(context.Background())

	connectFn := func() (net.Conn, error) {
		return net.Dial("tcp", address)
	}

	conn, err := connectFn()
	if err != nil {
		cancel()
		return nil
	}

	tc := &Connection{
		address:     address,
		connectFn:   connectFn,
		conn:        conn,
		state:       stateConnected,
		readBuf:     make([]byte, 4096),
		lastRead:    time.Now(),
		lastWrite:   time.Now(),
		idleTimeout: idleTimeout,
		ctx:         ctx,
		cancel:      cancel,
		poolSize:    1,
	}

	go tc.idleMonitor()

	return tc
}

func WrapConnection(conn net.Conn, idleTimeout time.Duration) *Connection {
	if idleTimeout == 0 {
		idleTimeout = defaultIdleTimeout
	}

	ctx, cancel := context.WithCancel(context.Background())

	tc := &Connection{
		conn:        conn,
		state:       stateConnected,
		readBuf:     make([]byte, 4096),
		lastRead:    time.Now(),
		lastWrite:   time.Now(),
		idleTimeout: idleTimeout,
		ctx:         ctx,
		cancel:      cancel,
		poolSize:    1,
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
	newConn, err := t.connectFn()
	if err != nil {
		return err
	}

	if t.conn != nil {
		t.conn.Close()
	}
	t.conn = newConn
	return nil
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

func (t *Connection) Send(data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.ensureConnected(); err != nil {
		return err
	}

	_, err := t.conn.Write(data)
	if err != nil {
		t.state = stateDisconnected
		return err
	}

	t.lastWrite = time.Now()
	return nil
}

func (t *Connection) Receive() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.ensureConnected(); err != nil {
		return nil, err
	}

	n, err := t.conn.Read(t.readBuf)
	if err != nil {
		t.state = stateDisconnected
		return nil, err
	}

	t.lastRead = time.Now()
	result := make([]byte, n)
	copy(result, t.readBuf[:n])
	return result, nil
}

func (t *Connection) close() error {
	if !atomic.CompareAndSwapInt32(&t.closed, 0, 1) {
		return nil
	}

	t.mu.Lock()
	t.state = stateDisconnected
	t.mu.Unlock()

	t.cancel()
	if t.conn != nil {
		t.conn.Close()
	}
	return nil
}

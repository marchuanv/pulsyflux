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
	conn        net.Conn // For wrapped connections only
	state       state
	mu          sync.RWMutex
	readBuf     []byte
	lastRead    time.Time
	lastWrite   time.Time
	idleTimeout time.Duration
	closed      int32
	inUse       int32
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewConnection(address string, id uuid.UUID) *Connection {
	ctx, cancel := context.WithCancel(context.Background())

	_, err := globalPool.getOrCreate(address)
	if err != nil {
		cancel()
		return nil
	}

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
	}

	go tc.idleMonitor()

	return tc
}

func WrapConnection(conn net.Conn, id uuid.UUID) *Connection {
	ctx, cancel := context.WithCancel(context.Background())

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
	if !atomic.CompareAndSwapInt32(&t.inUse, 0, 1) {
		return errConnectionInUse
	}
	defer atomic.StoreInt32(&t.inUse, 0)

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

	// Send chunks
	for offset := 0; offset < len(data); offset += chunkSize {
		end := offset + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[offset:end]
		chunkLen := uint32(len(chunk))

		// Frame: [id_len(1)][id][total_len(4)][chunk_len(4)][chunk]
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
	if !atomic.CompareAndSwapInt32(&t.inUse, 0, 1) {
		return nil, errConnectionInUse
	}
	defer atomic.StoreInt32(&t.inUse, 0)

	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.ensureConnected(); err != nil {
		return nil, err
	}

	var result []byte
	var totalLen uint32
	var received uint32

	for {
		// Read frame: [id_len(1)][id][total_len(4)][chunk_len(4)][chunk]
		idLenBuf := make([]byte, 1)
		if err := t.readFull(idLenBuf); err != nil {
			return nil, err
		}

		idLen := int(idLenBuf[0])
		idBuf := make([]byte, idLen)
		if err := t.readFull(idBuf); err != nil {
			return nil, err
		}

		receivedID := string(idBuf)
		if receivedID != t.id.String() {
			// Wrong connection ID, skip this chunk
			totalLenBuf := make([]byte, 4)
			if err := t.readFull(totalLenBuf); err != nil {
				return nil, err
			}
			chunkLenBuf := make([]byte, 4)
			if err := t.readFull(chunkLenBuf); err != nil {
				return nil, err
			}
			chunkLen := uint32(chunkLenBuf[0])<<24 | uint32(chunkLenBuf[1])<<16 | uint32(chunkLenBuf[2])<<8 | uint32(chunkLenBuf[3])
			skipBuf := make([]byte, chunkLen)
			if err := t.readFull(skipBuf); err != nil {
				return nil, err
			}
			continue
		}

		totalLenBuf := make([]byte, 4)
		if err := t.readFull(totalLenBuf); err != nil {
			return nil, err
		}
		totalLen = uint32(totalLenBuf[0])<<24 | uint32(totalLenBuf[1])<<16 | uint32(totalLenBuf[2])<<8 | uint32(totalLenBuf[3])

		chunkLenBuf := make([]byte, 4)
		if err := t.readFull(chunkLenBuf); err != nil {
			return nil, err
		}
		chunkLen := uint32(chunkLenBuf[0])<<24 | uint32(chunkLenBuf[1])<<16 | uint32(chunkLenBuf[2])<<8 | uint32(chunkLenBuf[3])

		chunk := make([]byte, chunkLen)
		if err := t.readFull(chunk); err != nil {
			return nil, err
		}

		if result == nil {
			result = make([]byte, 0, totalLen)
		}
		result = append(result, chunk...)
		received += chunkLen

		if received >= totalLen {
			t.lastRead = time.Now()
			return result, nil
		}
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

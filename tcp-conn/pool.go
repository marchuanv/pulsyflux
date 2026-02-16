package tcpconn

import (
	"context"
	"github.com/google/uuid"
	"net"
	"sync"
	"sync/atomic"
)

var globalPool = &connectionPool{
	pools: make(map[string]*physicalPool),
}

type connectionPool struct {
	pools map[string]*physicalPool
	mu    sync.RWMutex
}

type physicalPool struct {
	address  string
	conn     net.Conn
	mu       sync.RWMutex
	refCount int32
	routes   map[uuid.UUID]chan []byte
	routesMu sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

func (cp *connectionPool) getOrCreate(address string) (net.Conn, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if pool, exists := cp.pools[address]; exists {
		atomic.AddInt32(&pool.refCount, 1)
		return pool.conn, nil
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	pool := &physicalPool{
		address:  address,
		conn:     conn,
		refCount: 1,
		routes:   make(map[uuid.UUID]chan []byte),
		ctx:      ctx,
		cancel:   cancel,
	}

	cp.pools[address] = pool
	go pool.demux()

	return conn, nil
}

func (cp *connectionPool) release(address string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if pool, exists := cp.pools[address]; exists {
		if atomic.AddInt32(&pool.refCount, -1) == 0 {
			pool.cancel()
			pool.conn.Close()
			delete(cp.pools, address)
		}
	}
}

func (cp *connectionPool) get(address string) net.Conn {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if pool, exists := cp.pools[address]; exists {
		return pool.conn
	}
	return nil
}

func (cp *connectionPool) register(address string, id uuid.UUID, ch chan []byte) {
	cp.mu.RLock()
	pool, exists := cp.pools[address]
	cp.mu.RUnlock()

	if exists {
		pool.routesMu.Lock()
		pool.routes[id] = ch
		pool.routesMu.Unlock()
	}
}

func (cp *connectionPool) unregister(address string, id uuid.UUID) {
	cp.mu.RLock()
	pool, exists := cp.pools[address]
	cp.mu.RUnlock()

	if exists {
		pool.routesMu.Lock()
		if ch, ok := pool.routes[id]; ok {
			close(ch)
			delete(pool.routes, id)
		}
		pool.routesMu.Unlock()
	}
}

func (pp *physicalPool) demux() {
	messages := make(map[uuid.UUID]*messageAssembly)

	for {
		select {
		case <-pp.ctx.Done():
			return
		default:
		}

		// Read frame: [id_len(1)][id][total_len(4)][chunk_len(4)][chunk]
		idLenBuf := make([]byte, 1)
		if err := pp.readFull(idLenBuf); err != nil {
			return
		}

		idLen := int(idLenBuf[0])
		idBuf := make([]byte, idLen)
		if err := pp.readFull(idBuf); err != nil {
			return
		}

		id, err := uuid.Parse(string(idBuf))
		if err != nil {
			return
		}

		totalLenBuf := make([]byte, 4)
		if err := pp.readFull(totalLenBuf); err != nil {
			return
		}
		totalLen := uint32(totalLenBuf[0])<<24 | uint32(totalLenBuf[1])<<16 | uint32(totalLenBuf[2])<<8 | uint32(totalLenBuf[3])

		chunkLenBuf := make([]byte, 4)
		if err := pp.readFull(chunkLenBuf); err != nil {
			return
		}
		chunkLen := uint32(chunkLenBuf[0])<<24 | uint32(chunkLenBuf[1])<<16 | uint32(chunkLenBuf[2])<<8 | uint32(chunkLenBuf[3])

		chunk := make([]byte, chunkLen)
		if err := pp.readFull(chunk); err != nil {
			return
		}

		// Get or create message assembly
		assembly, exists := messages[id]
		if !exists {
			assembly = &messageAssembly{
				totalLen: totalLen,
				data:     make([]byte, 0, totalLen),
			}
			messages[id] = assembly
		}

		assembly.data = append(assembly.data, chunk...)

		// If message complete, send to channel
		if uint32(len(assembly.data)) >= assembly.totalLen {
			pp.routesMu.RLock()
			ch, exists := pp.routes[id]
			pp.routesMu.RUnlock()

			if exists {
				select {
				case ch <- assembly.data:
				case <-pp.ctx.Done():
					return
				}
			}
			delete(messages, id)
		}
	}
}

type messageAssembly struct {
	totalLen uint32
	data     []byte
}

func (pp *physicalPool) readFull(buf []byte) error {
	offset := 0
	for offset < len(buf) {
		n, err := pp.conn.Read(buf[offset:])
		if err != nil {
			return err
		}
		offset += n
	}
	return nil
}

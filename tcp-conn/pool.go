package tcpconn

import (
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

	cp.pools[address] = &physicalPool{
		address:  address,
		conn:     conn,
		refCount: 1,
	}

	return conn, nil
}

func (cp *connectionPool) release(address string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if pool, exists := cp.pools[address]; exists {
		if atomic.AddInt32(&pool.refCount, -1) == 0 {
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

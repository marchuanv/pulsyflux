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
	demuxer  *demuxer
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
		demuxer: &demuxer{
			conn:   conn,
			routes: make(map[uuid.UUID]chan []byte),
			ctx:    ctx,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	cp.pools[address] = pool
	go pool.demuxer.run()

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
		pool.demuxer.register(id, ch)
	}
}

func (cp *connectionPool) unregister(address string, id uuid.UUID) {
	cp.mu.RLock()
	pool, exists := cp.pools[address]
	cp.mu.RUnlock()

	if exists {
		pool.demuxer.unregister(id)
	}
}


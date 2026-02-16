package tcpconn

import (
	"context"
	"github.com/google/uuid"
	"net"
	"sync"
)

type messageAssembly struct {
	totalLen uint32
	data     []byte
}

type demuxer struct {
	conn     net.Conn
	routes   map[uuid.UUID]chan []byte
	routesMu sync.RWMutex
	ctx      context.Context
}

func (d *demuxer) register(id uuid.UUID, ch chan []byte) {
	d.routesMu.Lock()
	d.routes[id] = ch
	d.routesMu.Unlock()
}

func (d *demuxer) unregister(id uuid.UUID) {
	d.routesMu.Lock()
	if ch, ok := d.routes[id]; ok {
		close(ch)
		delete(d.routes, id)
	}
	d.routesMu.Unlock()
}

func (d *demuxer) run() {
	messages := make(map[uuid.UUID]*messageAssembly)

	for {
		select {
		case <-d.ctx.Done():
			return
		default:
		}

		idLenBuf := make([]byte, 1)
		if err := d.readFull(idLenBuf); err != nil {
			return
		}

		idLen := int(idLenBuf[0])
		idBuf := make([]byte, idLen)
		if err := d.readFull(idBuf); err != nil {
			return
		}

		id, err := uuid.Parse(string(idBuf))
		if err != nil {
			return
		}

		totalLenBuf := make([]byte, 4)
		if err := d.readFull(totalLenBuf); err != nil {
			return
		}
		totalLen := uint32(totalLenBuf[0])<<24 | uint32(totalLenBuf[1])<<16 | uint32(totalLenBuf[2])<<8 | uint32(totalLenBuf[3])

		chunkLenBuf := make([]byte, 4)
		if err := d.readFull(chunkLenBuf); err != nil {
			return
		}
		chunkLen := uint32(chunkLenBuf[0])<<24 | uint32(chunkLenBuf[1])<<16 | uint32(chunkLenBuf[2])<<8 | uint32(chunkLenBuf[3])

		chunk := make([]byte, chunkLen)
		if err := d.readFull(chunk); err != nil {
			return
		}

		assembly, exists := messages[id]
		if !exists {
			assembly = &messageAssembly{
				totalLen: totalLen,
				data:     make([]byte, 0, totalLen),
			}
			messages[id] = assembly
		}

		assembly.data = append(assembly.data, chunk...)

		if uint32(len(assembly.data)) >= assembly.totalLen {
			d.routesMu.RLock()
			ch, exists := d.routes[id]
			d.routesMu.RUnlock()

			if exists {
				select {
				case ch <- assembly.data:
				case <-d.ctx.Done():
					return
				}
			}
			delete(messages, id)
		}
	}
}

func (d *demuxer) readFull(buf []byte) error {
	offset := 0
	for offset < len(buf) {
		n, err := d.conn.Read(buf[offset:])
		if err != nil {
			return err
		}
		offset += n
	}
	return nil
}

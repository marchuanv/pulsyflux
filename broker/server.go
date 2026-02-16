package broker

import (
	"encoding/json"
	"net"
	tcpconn "pulsyflux/tcp-conn"
	"sync"

	"github.com/google/uuid"
)

type Message struct {
	Topic   string
	Payload []byte
}

type channel struct {
	id      uuid.UUID
	conns   []*tcpconn.Connection
	handled map[net.Conn]bool
	mu      sync.RWMutex
}

type Server struct {
	address  string
	listener net.Listener
	channels map[uuid.UUID]*channel
	mu       sync.RWMutex
	done     chan struct{}
}

func NewServer(address string) *Server {
	return &Server{
		address:  address,
		channels: make(map[uuid.UUID]*channel),
		done:     make(chan struct{}),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	s.listener = ln

	go s.acceptLoop()
	return nil
}

func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

func (s *Server) acceptLoop() {
	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			continue
		}

		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	control := tcpconn.WrapConnection(conn, controlUUID)

	for {
		data, err := control.Receive()
		if err != nil {
			return
		}
		channelID, _ := uuid.Parse(string(data))

		s.mu.Lock()
		ch := s.channels[channelID]
		if ch == nil {
			ch = &channel{
				id:      channelID,
				conns:   make([]*tcpconn.Connection, 0),
				handled: make(map[net.Conn]bool),
			}
			s.channels[channelID] = ch
		}
		if !ch.handled[conn] {
			ch.handled[conn] = true
			s.mu.Unlock()
			go s.handleChannel(conn, ch)
		} else {
			s.mu.Unlock()
		}
	}
}

func (s *Server) handleChannel(conn net.Conn, ch *channel) {
	c := tcpconn.WrapConnection(conn, ch.id)
	ch.mu.Lock()
	ch.conns = append(ch.conns, c)
	ch.mu.Unlock()

	for {
		data, err := c.Receive()
		if err != nil {
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		ch.broadcast(c, data)
	}
}

func (ch *channel) broadcast(sender *tcpconn.Connection, data []byte) {
	ch.mu.RLock()
	for _, conn := range ch.conns {
		if conn == sender {
			continue
		}
		conn.Send(data)
	}
	ch.mu.RUnlock()
}

func (s *Server) Stop() error {
	close(s.done)
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

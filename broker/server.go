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

type clientInfo struct {
	id      uuid.UUID
	wrapped *tcpconn.Connection
	sendConn *tcpconn.Connection
}

type channel struct {
	id      uuid.UUID
	clients map[uuid.UUID]*clientInfo
	mu      sync.RWMutex
}

type Server struct {
	address  string
	listener net.Listener
	channels map[uuid.UUID]*channel
	clients  map[net.Conn]*clientInfo
	mu       sync.RWMutex
	done     chan struct{}
}

func NewServer(address string) *Server {
	return &Server{
		address:  address,
		channels: make(map[uuid.UUID]*channel),
		clients:  make(map[net.Conn]*clientInfo),
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
	wrapped := tcpconn.WrapConnection(conn, uuid.UUID{})

	for {
		data, err := wrapped.Receive()
		if err != nil {
			return
		}

		var cmsg clientMessage
		if err := json.Unmarshal(data, &cmsg); err != nil {
			continue
		}

		clientID, err := uuid.Parse(cmsg.ClientID)
		if err != nil {
			continue
		}

		channelID, err := uuid.Parse(cmsg.ChannelID)
		if err != nil {
			continue
		}

		s.mu.Lock()
		client := s.clients[conn]
		if client == nil {
			sendConn := tcpconn.NewConnection(s.listener.Addr().String(), clientID)
			client = &clientInfo{
				id:       clientID,
				wrapped:  wrapped,
				sendConn: sendConn,
			}
			s.clients[conn] = client
		}

		ch := s.channels[channelID]
		if ch == nil {
			ch = &channel{
				id:      channelID,
				clients: make(map[uuid.UUID]*clientInfo),
			}
			s.channels[channelID] = ch
		}
		if ch.clients[clientID] == nil {
			ch.clients[clientID] = client
		}
		s.mu.Unlock()

		if len(cmsg.Payload) > 0 {
			msg := Message{
				Topic:   channelID.String(),
				Payload: cmsg.Payload,
			}
			msgData, _ := json.Marshal(msg)
			ch.broadcast(clientID, msgData)
		}
	}
}

func (ch *channel) broadcast(senderID uuid.UUID, data []byte) {
	ch.mu.RLock()
	for id, client := range ch.clients {
		if id == senderID {
			continue
		}
		client.sendConn.Send(data)
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

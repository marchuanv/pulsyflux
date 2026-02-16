package broker

import (
	"encoding/json"
	"net"
	tcpconn "pulsyflux/tcp-conn"
	"sync"

	"github.com/google/uuid"
)

var (
	GlobalControlUUID = uuid.MustParse("00000000-0000-0000-0000-000000000000")
)

type channel struct {
	clients map[uuid.UUID]*tcpconn.Connection
	mu      sync.RWMutex
}

type Server struct {
	address  string
	listener net.Listener
	channels map[uuid.UUID]*channel
	clients  map[uuid.UUID]net.Conn
	mu       sync.RWMutex
	done     chan struct{}
}

func NewServer(address string) *Server {
	return &Server{
		address:  address,
		channels: make(map[uuid.UUID]*channel),
		clients:  make(map[uuid.UUID]net.Conn),
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
	// Use global control UUID to receive control messages from all clients
	control := tcpconn.WrapConnection(conn, GlobalControlUUID)

	// Process control messages (only for establishing channels)
	for {
		data, err := control.Receive()
		if err != nil {
			return
		}

		var cmsg controlMessage
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
		// Create/get channel
		ch := s.channels[channelID]
		if ch == nil {
			ch = &channel{
				clients: make(map[uuid.UUID]*tcpconn.Connection),
			}
			s.channels[channelID] = ch
		}
		s.mu.Unlock()

		// Create channel connection and start handler
		ch.mu.Lock()
		if ch.clients[clientID] == nil {
			// Wrap with clientID to ensure each client has unique logical connection
			// even when multiple clients join the same channel
			channelConn := tcpconn.WrapConnection(conn, clientID)
			ch.clients[clientID] = channelConn
			go s.handleChannel(ch, clientID, channelConn)
		}
		ch.mu.Unlock()

		// Send ack to client
		control.Send([]byte{1})
	}
}

func (s *Server) handleChannel(ch *channel, clientID uuid.UUID, channelConn *tcpconn.Connection) {
	for {
		data, err := channelConn.Receive()
		if err != nil {
			return
		}
		ch.broadcast(clientID, data)
	}
}

func (ch *channel) broadcast(senderID uuid.UUID, data []byte) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	for clientID, channelConn := range ch.clients {
		if clientID == senderID {
			continue
		}
		channelConn.Send(data)
	}
}

func (s *Server) Stop() error {
	close(s.done)
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

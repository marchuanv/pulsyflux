package broker

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"sync"
	"time"
	tcpconn "pulsyflux/tcp-conn"
)

type Client struct {
	id       uuid.UUID
	address  string
	control  *tcpconn.Connection
	channels map[uuid.UUID]*tcpconn.Connection
	subs     map[uuid.UUID][]chan *Message
	mu       sync.RWMutex
}

type clientMessage struct {
	ClientID  string `json:"client_id"`
	ChannelID string `json:"channel_id"`
}

func NewClient(address string) (*Client, error) {
	clientID := uuid.New()
	// Use global control UUID for control connection
	control := tcpconn.NewConnection(address, GlobalControlUUID)
	if control == nil {
		return nil, fmt.Errorf("failed to create connection")
	}

	client := &Client{
		id:       clientID,
		address:  address,
		control:  control,
		channels: make(map[uuid.UUID]*tcpconn.Connection),
		subs:     make(map[uuid.UUID][]chan *Message),
	}

	return client, nil
}

func (c *Client) getChannelConn(channelID uuid.UUID) *tcpconn.Connection {
	c.mu.Lock()
	defer c.mu.Unlock()

	if conn := c.channels[channelID]; conn != nil {
		return conn
	}

	// First time: send control message to establish channel on server
	joinMsg := clientMessage{
		ClientID:  c.id.String(),
		ChannelID: channelID.String(),
	}
	data, _ := json.Marshal(joinMsg)
	c.control.Send(data)

	// Small delay to ensure server processes control message
	time.Sleep(100 * time.Millisecond)

	// Create channel connection with clientID (not channelID)
	// This ensures each client has a unique logical connection
	conn := tcpconn.NewConnection(c.address, c.id)
	if conn == nil {
		return nil
	}
	c.channels[channelID] = conn
	go c.receiveLoop(channelID, conn)
	return conn
}

func (c *Client) receiveLoop(channelID uuid.UUID, conn *tcpconn.Connection) {
	for {
		data, err := conn.Receive()
		if err != nil {
			return
		}
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		// Deliver to subscribers for this channel
		c.mu.RLock()
		subs := c.subs[channelID]
		for _, sub := range subs {
			select {
			case sub <- &msg:
			default:
			}
		}
		c.mu.RUnlock()
	}
}

func (c *Client) Publish(channelID uuid.UUID, topic string, payload []byte) error {
	// Get or create channel connection
	channelConn := c.getChannelConn(channelID)
	if channelConn == nil {
		return fmt.Errorf("failed to get channel connection")
	}

	msg := Message{
		Topic:   topic,
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return channelConn.Send(data)
}

func (c *Client) Subscribe(channelID uuid.UUID, topic string) (<-chan *Message, error) {
	ch := make(chan *Message, 100)
	c.mu.Lock()
	c.subs[channelID] = append(c.subs[channelID], ch)
	c.mu.Unlock()

	// Initialize channel (sends control message and creates connection)
	c.getChannelConn(channelID)

	return ch, nil
}

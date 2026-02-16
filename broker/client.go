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
	id        uuid.UUID
	channelID uuid.UUID
	address   string
	conn      *tcpconn.Connection
	subs      []chan []byte
	mu        sync.RWMutex
}

type controlMessage struct {
	ClientID  string `json:"client_id"`
	ChannelID string `json:"channel_id"`
}

func NewClient(address string, channelID uuid.UUID) (*Client, error) {
	clientID := uuid.New()
	control := tcpconn.NewConnection(address, GlobalControlUUID)
	if control == nil {
		return nil, fmt.Errorf("failed to create connection")
	}

	// Send control message to establish channel
	joinMsg := controlMessage{
		ClientID:  clientID.String(),
		ChannelID: channelID.String(),
	}
	data, _ := json.Marshal(joinMsg)
	control.Send(data)

	time.Sleep(100 * time.Millisecond)

	// Create channel connection
	conn := tcpconn.NewConnection(address, clientID)
	if conn == nil {
		return nil, fmt.Errorf("failed to create channel connection")
	}

	client := &Client{
		id:        clientID,
		channelID: channelID,
		address:   address,
		conn:      conn,
		subs:      []chan []byte{},
	}

	go client.receiveLoop()
	return client, nil
}



func (c *Client) receiveLoop() {
	for {
		data, err := c.conn.Receive()
		if err != nil {
			return
		}

		c.mu.RLock()
		for _, sub := range c.subs {
			select {
			case sub <- data:
			default:
			}
		}
		c.mu.RUnlock()
	}
}

func (c *Client) Publish(payload []byte) error {
	return c.conn.Send(payload)
}

func (c *Client) Subscribe() <-chan []byte {
	ch := make(chan []byte, 100)
	c.mu.Lock()
	c.subs = append(c.subs, ch)
	c.mu.Unlock()
	return ch
}

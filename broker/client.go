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
	subs      []chan *Message
	mu        sync.RWMutex
}

type clientMessage struct {
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
	joinMsg := clientMessage{
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
		subs:      []chan *Message{},
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
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		c.mu.RLock()
		for _, sub := range c.subs {
			select {
			case sub <- &msg:
			default:
			}
		}
		c.mu.RUnlock()
	}
}

func (c *Client) Publish(topic string, payload []byte) error {
	msg := Message{
		Topic:   topic,
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.conn.Send(data)
}

func (c *Client) Subscribe(topic string) <-chan *Message {
	ch := make(chan *Message, 100)
	c.mu.Lock()
	c.subs = append(c.subs, ch)
	c.mu.Unlock()
	return ch
}

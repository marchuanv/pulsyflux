package broker

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"sync"
	tcpconn "pulsyflux/tcp-conn"
)

type Client struct {
	id   uuid.UUID
	conn *tcpconn.Connection
	subs map[uuid.UUID][]chan *Message
	mu   sync.RWMutex
}

type clientMessage struct {
	ClientID  string `json:"client_id"`
	ChannelID string `json:"channel_id"`
	Payload   []byte `json:"payload"`
}

func NewClient(address string) (*Client, error) {
	clientID := uuid.New()
	conn := tcpconn.NewConnection(address, clientID)
	if conn == nil {
		return nil, fmt.Errorf("failed to create connection")
	}

	client := &Client{
		id:   clientID,
		conn: conn,
		subs: make(map[uuid.UUID][]chan *Message),
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

		channelID, err := uuid.Parse(msg.Topic)
		if err != nil {
			continue
		}

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
	msg := clientMessage{
		ClientID:  c.id.String(),
		ChannelID: channelID.String(),
		Payload:   payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.conn.Send(data)
}

func (c *Client) Subscribe(channelID uuid.UUID, topic string) (<-chan *Message, error) {
	ch := make(chan *Message, 100)
	c.mu.Lock()
	c.subs[channelID] = append(c.subs[channelID], ch)
	c.mu.Unlock()

	joinMsg := clientMessage{
		ClientID:  c.id.String(),
		ChannelID: channelID.String(),
		Payload:   []byte{},
	}
	data, _ := json.Marshal(joinMsg)
	c.conn.Send(data)

	return ch, nil
}

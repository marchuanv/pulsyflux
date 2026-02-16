package broker

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"sync"
	tcpconn "pulsyflux/tcp-conn"
)

type Client struct {
	address string
	control *tcpconn.Connection
	channels map[uuid.UUID]*tcpconn.Connection
	mu      sync.RWMutex
}

func NewClient(address string) (*Client, error) {
	control := tcpconn.NewConnection(address, uuid.New())
	if control == nil {
		return nil, fmt.Errorf("failed to create control connection")
	}
	return &Client{
		address:  address,
		control:  control,
		channels: make(map[uuid.UUID]*tcpconn.Connection),
	}, nil
}

func (c *Client) getChannel(channelID uuid.UUID) (*tcpconn.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.channels[channelID] == nil {
		c.control.Send([]byte(channelID.String()))
		c.channels[channelID] = tcpconn.NewConnection(c.address, channelID)
		if c.channels[channelID] == nil {
			return nil, fmt.Errorf("failed to create channel connection")
		}
	}
	return c.channels[channelID], nil
}

func (c *Client) Publish(channelID uuid.UUID, topic string, payload []byte) error {
	ch, err := c.getChannel(channelID)
	if err != nil {
		return err
	}
	msgID := uuid.New()
	ch.Send([]byte(msgID.String()))
	msg := Message{
		Topic:   topic,
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	message := tcpconn.NewConnection(c.address, msgID)
	if message == nil {
		return fmt.Errorf("failed to create message connection")
	}
	return message.Send(data)
}

func (c *Client) Subscribe(channelID uuid.UUID, topic string) (<-chan *Message, error) {
	ch, err := c.getChannel(channelID)
	if err != nil {
		return nil, err
	}
	msgID := uuid.New()
	ch.Send([]byte(msgID.String()))
	message := tcpconn.NewConnection(c.address, msgID)
	if message == nil {
		return nil, fmt.Errorf("failed to create message connection")
	}

	sub := &subscription{Channel: channelID, Topic: topic}
	data, err := json.Marshal(sub)
	if err != nil {
		return nil, err
	}
	if err := message.Send(data); err != nil {
		return nil, err
	}

	outCh := make(chan *Message, 100)
	go func() {
		defer close(outCh)
		for {
			data, err := message.Receive()
			if err != nil {
				return
			}
			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			outCh <- &msg
		}
	}()
	return outCh, nil
}

type subscription struct {
	Channel uuid.UUID
	Topic   string
}

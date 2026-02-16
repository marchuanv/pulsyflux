package broker

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"sync"
	tcpconn "pulsyflux/tcp-conn"
)

var controlID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type Client struct {
	address  string
	control  *tcpconn.Connection
	channels map[uuid.UUID]*channelConn
	mu       sync.RWMutex
}

type channelConn struct {
	conn *tcpconn.Connection
	subs []chan *Message
	mu   sync.RWMutex
}

func NewClient(address string) (*Client, error) {
	control := tcpconn.NewConnection(address, uuid.New())
	if control == nil {
		return nil, fmt.Errorf("failed to create control connection")
	}
	return &Client{
		address:  address,
		control:  control,
		channels: make(map[uuid.UUID]*channelConn),
	}, nil
}

func (c *Client) getChannel(channelID uuid.UUID) (*channelConn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.channels[channelID] == nil {
		c.control.Send([]byte(channelID.String()))
		conn := tcpconn.NewConnection(c.address, channelID)
		if conn == nil {
			return nil, fmt.Errorf("failed to create channel connection")
		}
		cc := &channelConn{
			conn: conn,
			subs: make([]chan *Message, 0),
		}
		c.channels[channelID] = cc
		go cc.receiveLoop()
	}
	return c.channels[channelID], nil
}

func (cc *channelConn) receiveLoop() {
	for {
		data, err := cc.conn.Receive()
		if err != nil {
			return
		}
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		cc.mu.RLock()
		for _, sub := range cc.subs {
			select {
			case sub <- &msg:
			default:
			}
		}
		cc.mu.RUnlock()
	}
}

func (c *Client) Publish(channelID uuid.UUID, topic string, payload []byte) error {
	cc, err := c.getChannel(channelID)
	if err != nil {
		return err
	}
	msg := Message{
		Topic:   topic,
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return cc.conn.Send(data)
}

func (c *Client) Subscribe(channelID uuid.UUID, topic string) (<-chan *Message, error) {
	cc, err := c.getChannel(channelID)
	if err != nil {
		return nil, err
	}

	ch := make(chan *Message, 100)
	cc.mu.Lock()
	cc.subs = append(cc.subs, ch)
	cc.mu.Unlock()
	return ch, nil
}

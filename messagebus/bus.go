package messagebus

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"pulsyflux/socket"
)

type messageBus struct {
	serverAddr string
	channelID  uuid.UUID
	consumer   *socket.Consumer
	provider   *socket.Provider
	channels   map[string][]chan *Message
	mu         sync.RWMutex
	done       chan struct{}
}

// NewBus creates a new message bus that can both publish and subscribe
func NewBus(serverAddr string, channelID uuid.UUID) (Bus, error) {
	consumer, err := socket.NewConsumer(serverAddr, channelID)
	if err != nil {
		return nil, err
	}

	provider, err := socket.NewProvider(serverAddr, channelID)
	if err != nil {
		consumer.Close()
		return nil, err
	}

	bus := &messageBus{
		serverAddr: serverAddr,
		channelID:  channelID,
		consumer:   consumer,
		provider:   provider,
		channels:   make(map[string][]chan *Message),
		done:       make(chan struct{}),
	}

	go bus.listen()
	return bus, nil
}

// Publish sends a message to all subscribers on the channel
func (b *messageBus) Publish(ctx context.Context, topic string, payload []byte, headers map[string]string) error {
	msg := &Message{
		ID:        uuid.New(),
		Topic:     topic,
		Payload:   payload,
		Headers:   headers,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	timeout := 5 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	_, err = b.consumer.Send(bytes.NewReader(data), timeout)
	return err
}

// Subscribe creates a channel for receiving messages on a topic
func (b *messageBus) Subscribe(topic string) (<-chan *Message, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan *Message, 100)
	b.channels[topic] = append(b.channels[topic], ch)
	return ch, nil
}

// Unsubscribe removes all channels for a topic
func (b *messageBus) Unsubscribe(topic string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range b.channels[topic] {
		close(ch)
	}
	delete(b.channels, topic)
	return nil
}

// Close shuts down the bus
func (b *messageBus) Close() error {
	close(b.done)
	b.consumer.Close()
	b.provider.Close()
	return nil
}

func (b *messageBus) listen() {
	for {
		select {
		case <-b.done:
			return
		default:
		}

		reqID, r, ok := b.provider.Receive()
		if !ok {
			return
		}

		go b.handleMessage(reqID, r)
	}
}

func (b *messageBus) handleMessage(reqID uuid.UUID, r io.Reader) {
	data, err := io.ReadAll(r)
	if err != nil {
		b.provider.Respond(reqID, nil, err)
		return
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		b.provider.Respond(reqID, nil, err)
		return
	}

	b.mu.RLock()
	channels := b.channels[msg.Topic]
	b.mu.RUnlock()

	if len(channels) > 0 {
		for _, ch := range channels {
			select {
			case ch <- &msg:
			default:
			}
		}
	}

	b.provider.Respond(reqID, bytes.NewReader([]byte("ok")), nil)
}
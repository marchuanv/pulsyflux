package broker

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

type Message struct {
	Topic   string
	Payload []byte
	Headers map[string]string
}

type Broker struct {
	serverAddr   string
	brokerChanID uuid.UUID
	provider     *socket.Provider
	publisher    *socket.Consumer
	subscribers  map[uuid.UUID]*socket.Provider // channelID -> Provider
	topics       map[string][]uuid.UUID         // topic -> channelIDs
	consumers    map[uuid.UUID]*socket.Consumer // channelID -> Consumer
	mu           sync.RWMutex
	done         chan struct{}
}

func NewBroker(serverAddr string, brokerChanID uuid.UUID) (*Broker, error) {
	provider, err := socket.NewProvider(serverAddr, brokerChanID)
	if err != nil {
		return nil, err
	}

	publisher, err := socket.NewConsumer(serverAddr, brokerChanID)
	if err != nil {
		provider.Close()
		return nil, err
	}

	b := &Broker{
		serverAddr:   serverAddr,
		brokerChanID: brokerChanID,
		provider:     provider,
		publisher:    publisher,
		subscribers:  make(map[uuid.UUID]*socket.Provider),
		topics:       make(map[string][]uuid.UUID),
		consumers:    make(map[uuid.UUID]*socket.Consumer),
		done:         make(chan struct{}),
	}

	go b.run()
	return b, nil
}

func (b *Broker) run() {
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

func (b *Broker) handleMessage(reqID uuid.UUID, r io.Reader) {
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

	// Respond immediately to publisher
	b.provider.Respond(reqID, bytes.NewReader([]byte("ok")), nil)

	b.mu.RLock()
	channelIDs := b.topics[msg.Topic]
	b.mu.RUnlock()

	// Forward to all subscribers using persistent consumers
	for _, chanID := range channelIDs {
		b.mu.RLock()
		consumer := b.consumers[chanID]
		b.mu.RUnlock()

		if consumer != nil {
			consumer.Send(bytes.NewReader(data), 5*time.Second)
		}
	}
}

func (b *Broker) Publish(ctx context.Context, topic string, payload []byte, headers map[string]string) error {
	msg := Message{
		Topic:   topic,
		Payload: payload,
		Headers: headers,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	timeout := 5 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	_, err = b.publisher.Send(bytes.NewReader(data), timeout)
	return err
}

func (b *Broker) Subscribe(topic string) (<-chan *Message, error) {
	channelID := uuid.New()
	provider, err := socket.NewProvider(b.serverAddr, channelID)
	if err != nil {
		return nil, err
	}

	consumer, err := socket.NewConsumer(b.serverAddr, channelID)
	if err != nil {
		provider.Close()
		return nil, err
	}

	ch := make(chan *Message, 100)

	b.mu.Lock()
	b.subscribers[channelID] = provider
	b.consumers[channelID] = consumer
	b.topics[topic] = append(b.topics[topic], channelID)
	b.mu.Unlock()

	// Start receiver goroutine
	go func() {
		for {
			_, r, ok := provider.Receive()
			if !ok {
				close(ch)
				return
			}

			data, err := io.ReadAll(r)
			if err != nil {
				continue
			}

			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}

			select {
			case ch <- &msg:
			default:
			}
		}
	}()

	return ch, nil
}



func (b *Broker) Unsubscribe(topic string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	channelIDs := b.topics[topic]
	delete(b.topics, topic)

	for _, channelID := range channelIDs {
		if provider, exists := b.subscribers[channelID]; exists {
			provider.Close()
			delete(b.subscribers, channelID)
		}
		if consumer, exists := b.consumers[channelID]; exists {
			consumer.Close()
			delete(b.consumers, channelID)
		}
	}
}

func (b *Broker) Close() error {
	close(b.done)

	b.mu.Lock()
	for _, provider := range b.subscribers {
		provider.Close()
	}
	for _, consumer := range b.consumers {
		consumer.Close()
	}
	b.mu.Unlock()

	b.publisher.Close()
	return b.provider.Close()
}

package messagebus

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Message represents a message in the bus
type Message struct {
	ID        uuid.UUID
	Topic     string
	Payload   []byte
	Headers   map[string]string
	Timestamp time.Time
}

// Handler processes messages
type Handler func(ctx context.Context, msg *Message) error

// Publisher publishes messages to topics
type Publisher interface {
	Publish(ctx context.Context, topic string, payload []byte, headers map[string]string) error
	Close() error
}

// Subscriber subscribes to topics
type Subscriber interface {
	Subscribe(topic string) (<-chan *Message, error)
	Unsubscribe(topic string) error
	Close() error
}

// Bus combines publisher and subscriber
type Bus interface {
	Publisher
	Subscriber
}

package broker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"pulsyflux/socket"
)

func TestPubSub(t *testing.T) {
	server := socket.NewServer("9100")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	brokerChanID := uuid.New()
	broker, err := NewBroker("127.0.0.1:9100", brokerChanID)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	time.Sleep(50 * time.Millisecond)

	ch, err := broker.Subscribe("test.topic")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	err = broker.Publish(ctx, "test.topic", []byte("hello"), map[string]string{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}

	msg := <-ch

	if msg.Topic != "test.topic" {
		t.Errorf("Expected topic test.topic, got %s", msg.Topic)
	}
	if string(msg.Payload) != "hello" {
		t.Errorf("Expected payload hello, got %s", string(msg.Payload))
	}
	if msg.Headers["key"] != "value" {
		t.Errorf("Expected header key=value, got %s", msg.Headers["key"])
	}
}

func TestMultipleSubscribers(t *testing.T) {
	server := socket.NewServer("9101")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	brokerChanID := uuid.New()
	broker, err := NewBroker("127.0.0.1:9101", brokerChanID)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	time.Sleep(50 * time.Millisecond)

	ch1, _ := broker.Subscribe("test.topic")
	ch2, _ := broker.Subscribe("test.topic")

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	broker.Publish(ctx, "test.topic", []byte("broadcast"), nil)

	msg1 := <-ch1
	msg2 := <-ch2

	if string(msg1.Payload) != "broadcast" {
		t.Errorf("Sub1 expected broadcast, got %s", string(msg1.Payload))
	}
	if string(msg2.Payload) != "broadcast" {
		t.Errorf("Sub2 expected broadcast, got %s", string(msg2.Payload))
	}
}

func TestUnsubscribe(t *testing.T) {
	server := socket.NewServer("9102")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	brokerChanID := uuid.New()
	broker, err := NewBroker("127.0.0.1:9102", brokerChanID)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	time.Sleep(50 * time.Millisecond)

	ch, _ := broker.Subscribe("test.topic")

	time.Sleep(50 * time.Millisecond)

	broker.Unsubscribe("test.topic")

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	broker.Publish(ctx, "test.topic", []byte("should not receive"), nil)

	select {
	case msg, ok := <-ch:
		if ok {
			t.Fatalf("Should not receive message after unsubscribe, got: %s", string(msg.Payload))
		}
		// Channel closed, expected
	case <-time.After(500 * time.Millisecond):
		// Expected - no message received
	}
}

func TestMultipleTopics(t *testing.T) {
	server := socket.NewServer("9103")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	brokerChanID := uuid.New()
	broker, err := NewBroker("127.0.0.1:9103", brokerChanID)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	time.Sleep(50 * time.Millisecond)

	ch1, _ := broker.Subscribe("topic.a")
	ch2, _ := broker.Subscribe("topic.b")

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	broker.Publish(ctx, "topic.a", []byte("message-a"), nil)
	broker.Publish(ctx, "topic.b", []byte("message-b"), nil)

	msg1 := <-ch1
	msg2 := <-ch2

	if string(msg1.Payload) != "message-a" {
		t.Errorf("Sub1 expected message-a, got %s", string(msg1.Payload))
	}
	if string(msg2.Payload) != "message-b" {
		t.Errorf("Sub2 expected message-b, got %s", string(msg2.Payload))
	}
}

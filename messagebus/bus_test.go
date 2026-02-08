package messagebus

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPubSub(t *testing.T) {
	server := NewServer("9091")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	bus, err := NewBus("127.0.0.1:9091", channelID)
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	ch, err := bus.Subscribe("test.topic")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	err = bus.Publish(ctx, "test.topic", []byte("hello"), map[string]string{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case msg := <-ch:
		if msg.Topic != "test.topic" {
			t.Errorf("Expected topic test.topic, got %s", msg.Topic)
		}
		if string(msg.Payload) != "hello" {
			t.Errorf("Expected payload hello, got %s", string(msg.Payload))
		}
		if msg.Headers["key"] != "value" {
			t.Errorf("Expected header key=value, got %s", msg.Headers["key"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	server := NewServer("9092")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	bus, err := NewBus("127.0.0.1:9092", channelID)
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	ch1, _ := bus.Subscribe("test.topic")
	ch2, _ := bus.Subscribe("test.topic")

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	bus.Publish(ctx, "test.topic", []byte("test"), nil)

	count := 0
	timeout := time.After(1 * time.Second)
	for count < 2 {
		select {
		case <-ch1:
			count++
		case <-ch2:
			count++
		case <-timeout:
			t.Fatalf("Expected 2 messages, got %d", count)
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	server := NewServer("9093")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	bus, err := NewBus("127.0.0.1:9093", channelID)
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	ch, _ := bus.Subscribe("test.topic")

	time.Sleep(50 * time.Millisecond)

	bus.Unsubscribe("test.topic")

	ctx := context.Background()
	bus.Publish(ctx, "test.topic", []byte("test"), nil)

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("Should not receive message after unsubscribe")
		}
		// Channel closed, expected
	case <-time.After(500 * time.Millisecond):
		// No message, also acceptable
	}
}

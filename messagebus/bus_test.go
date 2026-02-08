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
	
	// Create two separate buses on same channel
	bus1, err := NewBus("127.0.0.1:9091", channelID)
	if err != nil {
		t.Fatal(err)
	}
	defer bus1.Close()

	bus2, err := NewBus("127.0.0.1:9091", channelID)
	if err != nil {
		t.Fatal(err)
	}
	defer bus2.Close()

	// Subscribe on bus2
	ch, err := bus2.Subscribe("test.topic")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	// Publish from bus1
	ctx := context.Background()
	err = bus1.Publish(ctx, "test.topic", []byte("hello"), map[string]string{"key": "value"})
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
	
	// Publisher bus
	pubBus, err := NewBus("127.0.0.1:9092", channelID)
	if err != nil {
		t.Fatal(err)
	}
	defer pubBus.Close()

	// Two subscriber buses
	sub1, err := NewBus("127.0.0.1:9092", channelID)
	if err != nil {
		t.Fatal(err)
	}
	defer sub1.Close()

	sub2, err := NewBus("127.0.0.1:9092", channelID)
	if err != nil {
		t.Fatal(err)
	}
	defer sub2.Close()

	ch1, _ := sub1.Subscribe("test.topic")
	ch2, _ := sub2.Subscribe("test.topic")

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	pubBus.Publish(ctx, "test.topic", []byte("test"), nil)

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
	
	pubBus, err := NewBus("127.0.0.1:9093", channelID)
	if err != nil {
		t.Fatal(err)
	}
	defer pubBus.Close()

	subBus, err := NewBus("127.0.0.1:9093", channelID)
	if err != nil {
		t.Fatal(err)
	}
	defer subBus.Close()

	ch, _ := subBus.Subscribe("test.topic")

	time.Sleep(50 * time.Millisecond)

	subBus.Unsubscribe("test.topic")

	ctx := context.Background()
	pubBus.Publish(ctx, "test.topic", []byte("test"), nil)

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("Should not receive message after unsubscribe")
		}
	case <-time.After(500 * time.Millisecond):
	}
}

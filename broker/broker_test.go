package broker

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBasicPubSub(t *testing.T) {
	server := NewServer(":0")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop()

	addr := server.Addr()
	channelID := uuid.New()

	client1, _ := NewClient(addr, channelID)
	client2, _ := NewClient(addr, channelID)

	ch1 := client1.Subscribe("test")
	ch2 := client2.Subscribe("test")

	time.Sleep(500 * time.Millisecond)

	client1.Publish("test", []byte("hello"))

	msg := <-ch2
	if string(msg.Payload) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(msg.Payload))
	}

	select {
	case <-ch1:
		t.Error("sender should not receive own message")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestMultipleChannels(t *testing.T) {
	server := NewServer(":0")
	server.Start()
	defer server.Stop()

	addr := server.Addr()
	ch1ID := uuid.New()
	ch2ID := uuid.New()

	client1, _ := NewClient(addr, ch1ID)
	client2, _ := NewClient(addr, ch2ID)

	sub1 := client1.Subscribe("test")
	sub2 := client2.Subscribe("test")

	time.Sleep(500 * time.Millisecond)

	client1.Publish("test", []byte("channel1"))
	client2.Publish("test", []byte("channel2"))

	// Client1 should receive its own channel's message
	select {
	case msg := <-sub1:
		if string(msg.Payload) != "channel1" {
			t.Errorf("client1 expected 'channel1', got '%s'", string(msg.Payload))
		}
	case <-time.After(1 * time.Second):
		t.Error("client1 did not receive message from its channel")
	}

	// Client2 should receive its own channel's message
	select {
	case msg := <-sub2:
		if string(msg.Payload) != "channel2" {
			t.Errorf("client2 expected 'channel2', got '%s'", string(msg.Payload))
		}
	case <-time.After(1 * time.Second):
		t.Error("client2 did not receive message from its channel")
	}
}

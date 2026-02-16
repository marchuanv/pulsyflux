package broker

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestBasicPubSub(t *testing.T) {
	server := NewServer(":0")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop()

	addr := server.Addr()
	channelID := uuid.New()

	client1, _ := NewClient(addr)
	client2, _ := NewClient(addr)

	ch1, _ := client1.Subscribe(channelID, "test")
	ch2, _ := client2.Subscribe(channelID, "test")

	time.Sleep(500 * time.Millisecond)

	client1.Publish(channelID, "test", []byte("hello"))

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

	client1, _ := NewClient(addr)
	client2, _ := NewClient(addr)

	sub1, _ := client1.Subscribe(ch1ID, "test")
	sub2, _ := client2.Subscribe(ch2ID, "test")

	time.Sleep(500 * time.Millisecond)

	client1.Publish(ch1ID, "test", []byte("channel1"))
	client2.Publish(ch2ID, "test", []byte("channel2"))

	select {
	case <-sub2:
		t.Error("channel2 should not receive channel1 message")
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case <-sub1:
		t.Error("channel1 should not receive channel2 message")
	case <-time.After(100 * time.Millisecond):
	}
}

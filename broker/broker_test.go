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

	ch1 := client1.Subscribe()
	ch2 := client2.Subscribe()

	time.Sleep(500 * time.Millisecond)

	client1.Publish([]byte("hello"))

	msg := <-ch2
	if string(msg) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(msg))
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
	client2, _ := NewClient(addr, ch1ID)
	client3, _ := NewClient(addr, ch2ID)
	client4, _ := NewClient(addr, ch2ID)

	_ = client1.Subscribe()
	sub2 := client2.Subscribe()
	_ = client3.Subscribe()
	sub4 := client4.Subscribe()

	time.Sleep(500 * time.Millisecond)

	client1.Publish([]byte("channel1"))
	client3.Publish([]byte("channel2"))

	// Client2 should receive channel1 message
	select {
	case msg := <-sub2:
		if string(msg) != "channel1" {
			t.Errorf("client2 expected 'channel1', got '%s'", string(msg))
		}
	case <-time.After(1 * time.Second):
		t.Error("client2 did not receive message from channel1")
	}

	// Client4 should receive channel2 message
	select {
	case msg := <-sub4:
		if string(msg) != "channel2" {
			t.Errorf("client4 expected 'channel2', got '%s'", string(msg))
		}
	case <-time.After(1 * time.Second):
		t.Error("client4 did not receive message from channel2")
	}

	// Client2 should NOT receive channel2 message
	select {
	case <-sub2:
		t.Error("client2 should not receive message from channel2")
	case <-time.After(100 * time.Millisecond):
	}

	// Client4 should NOT receive channel1 message
	select {
	case <-sub4:
		t.Error("client4 should not receive message from channel1")
	case <-time.After(100 * time.Millisecond):
	}
}

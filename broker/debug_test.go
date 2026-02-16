package broker

import (
	"fmt"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestDebug(t *testing.T) {
	server := NewServer(":0")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop()

	addr := server.Addr()
	fmt.Println("Server started at:", addr)

	channelID := uuid.New()
	fmt.Println("Channel ID:", channelID)

	client1, err := NewClient(addr)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Client1 created")

	client2, err := NewClient(addr)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Client2 created")

	ch1, err := client1.Subscribe(channelID, "test")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Client1 subscribed")

	ch2, err := client2.Subscribe(channelID, "test")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Client2 subscribed")

	time.Sleep(100 * time.Millisecond)

	fmt.Println("Client1 publishing...")
	err = client1.Publish(channelID, "test", []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Published")

	select {
	case msg := <-ch2:
		fmt.Printf("Client2 received: %s\n", string(msg.Payload))
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message on client2")
	}

	select {
	case <-ch1:
		t.Fatal("Client1 should not receive own message")
	case <-time.After(100 * time.Millisecond):
		fmt.Println("Client1 correctly did not receive own message")
	}
}

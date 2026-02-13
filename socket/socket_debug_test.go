package socket

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDebugFlow(t *testing.T) {
	server := NewServer("9095")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	
	client1, err := NewClient("127.0.0.1:9095", channelID)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()
	fmt.Println("Client1 created:", client1.clientID)

	client2, err := NewClient("127.0.0.1:9095", channelID)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()
	fmt.Println("Client2 created:", client2.clientID)

	time.Sleep(200 * time.Millisecond)

	// Client1 sends
	fmt.Println("Client1 sending request...")
	go func() {
		_, err := client1.Send(strings.NewReader("hello"), 5*time.Second)
		if err != nil {
			fmt.Println("Client1 send error:", err)
		} else {
			fmt.Println("Client1 send completed")
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Client2 receives
	fmt.Println("Client2 calling Receive()...")
	frame, err := client2.Receive()
	if err != nil {
		t.Fatalf("Client2 receive failed: %v", err)
	}
	fmt.Printf("Client2 received frame: Type=%d, Flags=%d\n", frame.Type, frame.Flags)

	if frame.Type != startFrame {
		t.Fatalf("Expected startFrame, got %d", frame.Type)
	}

	fmt.Println("Test passed!")
}

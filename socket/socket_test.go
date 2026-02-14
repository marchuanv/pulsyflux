package socket

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSendReceive(t *testing.T) {
	server := NewServer("9090")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, err := NewClient("127.0.0.1:9090", channelID)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()

	client2, err := NewClient("127.0.0.1:9090", channelID)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	incoming := make(chan io.Reader)
	outgoing := make(chan io.Reader)

	go func() {
		fmt.Println("Client2: Starting Receive")
		if err := client2.Receive(incoming, outgoing, 5*time.Second); err != nil {
			t.Errorf("Client2 receive failed: %v", err)
		}
		fmt.Println("Client2: Receive completed")
	}()

	go func() {
		fmt.Println("Handler: Waiting for incoming request")
		req := <-incoming
		data, _ := io.ReadAll(req)
		fmt.Printf("Handler: Received request: %q\n", string(data))
		if string(data) != "hello" {
			t.Errorf("Expected 'hello', got %q", string(data))
		}
		fmt.Println("Handler: Sending response")
		outgoing <- strings.NewReader("echo")
	}()

	time.Sleep(100 * time.Millisecond)

	fmt.Println("Client1: Sending request")
	response, err := client1.Send(strings.NewReader("hello"), 5*time.Second)
	if err != nil {
		t.Fatalf("Client1 send failed: %v", err)
	}

	data, _ := io.ReadAll(response)
	fmt.Printf("Client1: Received response: %q\n", string(data))
	if string(data) != "echo" {
		t.Errorf("Expected 'echo', got %q", string(data))
	}
}

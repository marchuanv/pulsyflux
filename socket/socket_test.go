package socket

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSendReceiveRespond(t *testing.T) {
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

	// Client2 receives and responds
	go func() {
		client2.Respond(strings.NewReader("echo"), 5*time.Second)
	}()

	time.Sleep(100 * time.Millisecond)

	// Client1 sends request
	response, err := client1.Send(strings.NewReader("hello"), 5*time.Second)
	if err != nil {
		t.Fatalf("Client1 send failed: %v", err)
	}

	data, _ := io.ReadAll(response)
	expected := "echo"
	if string(data) != expected {
		t.Errorf("Expected %q, got %q", expected, string(data))
	}
}

func TestNoOtherClients(t *testing.T) {
	server := NewServer("9091")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client, err := NewClient("127.0.0.1:9091", channelID)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Send request - should go to own queue
	go func() {
		time.Sleep(100 * time.Millisecond)
		client.Send(strings.NewReader("self"), 5*time.Second)
	}()

	// Receive own request
	frame, err := client.Receive()
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if frame.Type != startFrame {
		t.Errorf("Expected startFrame, got %d", frame.Type)
	}
}

func TestMultiplePeers(t *testing.T) {
	server := NewServer("9092")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9092", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9092", channelID)
	defer client2.Close()
	client3, _ := NewClient("127.0.0.1:9092", channelID)
	defer client3.Close()

	done := make(chan bool, 2)

	// Client2 and Client3 receive and respond
	go func() {
		frame, _ := client2.Receive()
		if frame.Type == startFrame {
			client2.Respond(strings.NewReader("from2"), 5*time.Second)
			done <- true
		}
	}()

	go func() {
		frame, _ := client3.Receive()
		if frame.Type == startFrame {
			client3.Respond(strings.NewReader("from3"), 5*time.Second)
			done <- true
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Client1 sends - should be enqueued to both peers
	resp, err := client1.Send(strings.NewReader("test"), 5*time.Second)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	data, _ := io.ReadAll(resp)
	if len(data) == 0 {
		t.Error("Expected response from peer")
	}

	<-done
}

func TestReceiveFromPeerQueue(t *testing.T) {
	server := NewServer("9093")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9093", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9093", channelID)
	defer client2.Close()

	// Client1 sends request
	go func() {
		client1.Send(strings.NewReader("request"), 5*time.Second)
	}()

	time.Sleep(100 * time.Millisecond)

	// Client2 receives from its own queue (which has client1's request)
	frame, err := client2.Receive()
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if frame.Type != startFrame {
		t.Errorf("Expected startFrame, got %d", frame.Type)
	}
}

func TestConcurrentRequests(t *testing.T) {
	server := NewServer("9094")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9094", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9094", channelID)
	defer client2.Close()

	done := make(chan bool, 3)

	// Client2 handles multiple requests
	for i := 0; i < 3; i++ {
		go func() {
			frame, err := client2.Receive()
			if err == nil && frame.Type == startFrame {
				client2.Respond(strings.NewReader("response"), 5*time.Second)
				done <- true
			}
		}()
	}

	time.Sleep(100 * time.Millisecond)

	// Client1 sends multiple requests
	for i := 0; i < 3; i++ {
		go func() {
			client1.Send(strings.NewReader("request"), 5*time.Second)
		}()
	}

	// Wait for all responses
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for responses")
		}
	}
}

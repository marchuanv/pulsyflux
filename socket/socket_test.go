package socket

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestClientBidirectional(t *testing.T) {
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
	go func() {
		client2.Respond(strings.NewReader("echo"), 5*time.Second)
	}()
	response, err := client1.Send(strings.NewReader("hello"), 20*time.Second)
	if err != nil {
		t.Fatalf("Client1 send failed: %v", err)
	}
	data, _ := io.ReadAll(response)
	expected := "echo"
	if string(data) != expected {
		t.Errorf("Expected %q, got %q", expected, string(data))
	}
}

func TestMultipleConsumersOneProvider(t *testing.T) {
	t.Skip("Test needs to be redesigned without Receive()")
}

func TestNoPeerAvailable(t *testing.T) {
	server := NewServer("9092")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()

	consumer, err := NewClient("127.0.0.1:9092", channelID)
	if err == nil {
		consumer.Close()
		t.Fatal("Expected handshake to fail when no peer available")
	}
}

func TestLargePayload(t *testing.T) {
	t.Skip("Test needs to be redesigned without Receive()")
}

func TestConcurrentChannels(t *testing.T) {
	t.Skip("Test needs to be redesigned without Receive()")
}

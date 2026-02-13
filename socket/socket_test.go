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

func TestMultiplePeers(t *testing.T) {
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
	client3, _ := NewClient("127.0.0.1:9093", channelID)
	defer client3.Close()

	done := make(chan string, 2)
	go func() {
		client2.Respond(strings.NewReader("from2"), 5*time.Second)
		done <- "client2"
	}()
	go func() {
		client3.Respond(strings.NewReader("from3"), 5*time.Second)
		done <- "client3"
	}()

	time.Sleep(100 * time.Millisecond)
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

func TestBroadcast(t *testing.T) {
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
	client3, _ := NewClient("127.0.0.1:9094", channelID)
	defer client3.Close()

	received := make(chan bool, 2)
	go func() {
		client2.Respond(strings.NewReader("ack2"), 5*time.Second)
		received <- true
	}()
	go func() {
		client3.Respond(strings.NewReader("ack3"), 5*time.Second)
		received <- true
	}()

	time.Sleep(100 * time.Millisecond)
	err := client1.Broadcast(strings.NewReader("broadcast"), 5*time.Second)
	if err != nil {
		t.Fatalf("Broadcast failed: %v", err)
	}
	<-received
	<-received
}

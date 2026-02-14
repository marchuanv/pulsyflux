package socket

import (
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
		if err := client2.Receive(incoming, outgoing, 5*time.Second); err != nil {
			t.Errorf("Client2 receive failed: %v", err)
		}
	}()

	go func() {
		req := <-incoming
		data, _ := io.ReadAll(req)
		if string(data) != "hello" {
			t.Errorf("Expected 'hello', got %q", string(data))
		}
		outgoing <- strings.NewReader("echo")
	}()

	time.Sleep(100 * time.Millisecond)

	response, err := client1.Send(strings.NewReader("hello"), 5*time.Second)
	if err != nil {
		t.Fatalf("Client1 send failed: %v", err)
	}

	data, _ := io.ReadAll(response)
	if string(data) != "echo" {
		t.Errorf("Expected 'echo', got %q", string(data))
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

	incoming := make(chan io.Reader)
	outgoing := make(chan io.Reader)

	go func() {
		time.Sleep(100 * time.Millisecond)
		client.Send(strings.NewReader("self"), 5*time.Second)
	}()

	go func() {
		if err := client.Receive(incoming, outgoing, 5*time.Second); err != nil {
			t.Errorf("Receive failed: %v", err)
		}
	}()

	select {
	case req := <-incoming:
		data, _ := io.ReadAll(req)
		if string(data) != "self" {
			t.Errorf("Expected 'self', got %q", string(data))
		}
		outgoing <- strings.NewReader("response")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for self request")
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

	incoming2 := make(chan io.Reader)
	outgoing2 := make(chan io.Reader)
	incoming3 := make(chan io.Reader)
	outgoing3 := make(chan io.Reader)

	go func() {
		client2.Receive(incoming2, outgoing2, 5*time.Second)
	}()

	go func() {
		<-incoming2
		outgoing2 <- strings.NewReader("from2")
	}()

	go func() {
		client3.Receive(incoming3, outgoing3, 5*time.Second)
	}()

	go func() {
		<-incoming3
		outgoing3 <- strings.NewReader("from3")
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

	for i := 0; i < 3; i++ {
		go func() {
			incoming := make(chan io.Reader)
			outgoing := make(chan io.Reader)
			go func() {
				<-incoming
				outgoing <- strings.NewReader("response")
			}()
			if err := client2.Receive(incoming, outgoing, 5*time.Second); err == nil {
				done <- true
			}
		}()
	}

	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 3; i++ {
		go func() {
			client1.Send(strings.NewReader("request"), 5*time.Second)
		}()
	}

	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for responses")
		}
	}
}

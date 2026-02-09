package socket

import (
	"bytes"
	"io"
	"strings"
	"sync"
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

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()

	// Create provider client
	provider, err := NewClient("127.0.0.1:9090", channelID, roleProvider)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create consumer client
	consumer, err := NewClient("127.0.0.1:9090", channelID, roleConsumer)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Handle requests in background
	go func() {
		for {
			reqID, r, ok := provider.Receive()
			if !ok {
				break
			}
			data, _ := io.ReadAll(r)
			provider.Respond(reqID, bytes.NewReader([]byte("echo: "+string(data))), 5*time.Second)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Send request
	response, err := consumer.Send(strings.NewReader("hello"), 20*time.Second)
	if err != nil {
		t.Fatalf("Consumer send failed: %v", err)
	}

	data, _ := io.ReadAll(response)
	expected := "echo: hello"
	if string(data) != expected {
		t.Errorf("Expected %q, got %q", expected, string(data))
	}
}

func TestMultipleConsumersOneProvider(t *testing.T) {
	server := NewServer("9091")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()

	provider, _ := NewClient("127.0.0.1:9091", channelID, roleProvider)
	defer provider.Close()

	go func() {
		for {
			reqID, _, ok := provider.Receive()
			if !ok {
				break
			}
			provider.Respond(reqID, strings.NewReader("processed"), 5*time.Second)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			consumer, _ := NewClient("127.0.0.1:9091", channelID, roleConsumer)
			defer consumer.Close()

			resp, err := consumer.Send(strings.NewReader("request"), 2*time.Second)
			if err != nil {
				t.Errorf("Consumer %d error: %v", id, err)
				return
			}
			data, _ := io.ReadAll(resp)
			if string(data) != "processed" {
				t.Errorf("Consumer %d: expected 'processed', got %q", id, string(data))
			}
		}(i)
	}

	wg.Wait()
}

func TestNoPeerAvailable(t *testing.T) {
	server := NewServer("9092")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()

	consumer, err := NewClient("127.0.0.1:9092", channelID, roleConsumer)
	if err == nil {
		consumer.Close()
		t.Fatal("Expected handshake to fail when no peer available")
	}
}

func TestLargePayload(t *testing.T) {
	server := NewServer("9093")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()

	provider, _ := NewClient("127.0.0.1:9093", channelID, roleProvider)
	defer provider.Close()

	go func() {
		for {
			reqID, _, ok := provider.Receive()
			if !ok {
				break
			}
			provider.Respond(reqID, strings.NewReader("received"), 5*time.Second)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewClient("127.0.0.1:9093", channelID, roleConsumer)
	defer consumer.Close()

	largeData := strings.Repeat("X", 2*1024*1024)
	resp, err := consumer.Send(strings.NewReader(largeData), 3*time.Second)
	if err != nil {
		t.Fatalf("Large payload failed: %v", err)
	}
	data, _ := io.ReadAll(resp)
	if string(data) != "received" {
		t.Errorf("Expected 'received', got %q", string(data))
	}
}

func TestConcurrentChannels(t *testing.T) {
	server := NewServer("9094")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	numChannels := 10
	var wg sync.WaitGroup
	wg.Add(numChannels)

	for i := 0; i < numChannels; i++ {
		go func(id int) {
			defer wg.Done()
			channelID := uuid.New()

			provider, _ := NewClient("127.0.0.1:9094", channelID, roleProvider)
			defer provider.Close()

			go func() {
				for {
					reqID, _, ok := provider.Receive()
					if !ok {
						break
					}
					provider.Respond(reqID, strings.NewReader("ok"), 5*time.Second)
				}
			}()

			time.Sleep(50 * time.Millisecond)

			consumer, _ := NewClient("127.0.0.1:9094", channelID, roleConsumer)
			defer consumer.Close()

			resp, err := consumer.Send(strings.NewReader("test"), 2*time.Second)
			if err != nil {
				t.Errorf("Channel %d error: %v", id, err)
				return
			}
			data, _ := io.ReadAll(resp)
			if string(data) != "ok" {
				t.Errorf("Channel %d: expected 'ok', got %q", id, string(data))
			}
		}(i)
	}

	wg.Wait()
}

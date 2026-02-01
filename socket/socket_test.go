package socket

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestConsumerProviderBidirectional(t *testing.T) {
	server := NewServer("9090")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()

	// Create provider that echoes back the request
	provider, err := NewProvider("127.0.0.1:9090", channelID)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Handle requests in background
	go func() {
		for {
			reqID, r, ok := provider.Receive()
			if !ok {
				break
			}
			data, _ := io.ReadAll(r)
			provider.Respond(reqID, []byte("echo: "+string(data)), nil)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Create consumer
	consumer, err := NewConsumer("127.0.0.1:9090", channelID)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

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

	provider, _ := NewProvider("127.0.0.1:9091", channelID)
	defer provider.Close()

	go func() {
		for {
			reqID, _, ok := provider.Receive()
			if !ok {
				break
			}
			provider.Respond(reqID, []byte("processed"), nil)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			consumer, _ := NewConsumer("127.0.0.1:9091", channelID)
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

	consumer, err := NewConsumer("127.0.0.1:9092", channelID)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	_, err = consumer.Send(strings.NewReader("no peer"), 1*time.Second)
	if err == nil {
		t.Fatal("Expected error when no peer available")
	}
	if !strings.Contains(err.Error(), "peer disappeared") {
		t.Fatalf("Expected 'peer disappeared' error, got: %v", err)
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

	provider, _ := NewProvider("127.0.0.1:9093", channelID)
	defer provider.Close()

	go func() {
		for {
			reqID, _, ok := provider.Receive()
			if !ok {
				break
			}
			provider.Respond(reqID, []byte("received"), nil)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewConsumer("127.0.0.1:9093", channelID)
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

			provider, _ := NewProvider("127.0.0.1:9094", channelID)
			defer provider.Close()

			go func() {
				for {
					reqID, _, ok := provider.Receive()
					if !ok {
						break
					}
					provider.Respond(reqID, []byte("ok"), nil)
				}
			}()

			time.Sleep(50 * time.Millisecond)

			consumer, _ := NewConsumer("127.0.0.1:9094", channelID)
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

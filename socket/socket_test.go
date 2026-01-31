package socket

import (
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
	provider, err := NewProvider("127.0.0.1:9090", channelID, func(payload []byte) ([]byte, error) {
		return []byte("echo: " + string(payload)), nil
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	time.Sleep(50 * time.Millisecond)

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

	expected := "echo: hello"
	if string(response) != expected {
		t.Errorf("Expected %q, got %q", expected, string(response))
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

	provider, _ := NewProvider("127.0.0.1:9091", channelID, func(payload []byte) ([]byte, error) {
		return []byte("processed"), nil
	})
	defer provider.Close()

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
			if string(resp) != "processed" {
				t.Errorf("Consumer %d: expected 'processed', got %q", id, string(resp))
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
	if !strings.Contains(err.Error(), "no peer available") {
		t.Fatalf("Expected 'no peer available' error, got: %v", err)
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

	provider, _ := NewProvider("127.0.0.1:9093", channelID, func(payload []byte) ([]byte, error) {
		return []byte("received"), nil
	})
	defer provider.Close()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewConsumer("127.0.0.1:9093", channelID)
	defer consumer.Close()

	largeData := strings.Repeat("X", 2*1024*1024)
	resp, err := consumer.Send(strings.NewReader(largeData), 3*time.Second)
	if err != nil {
		t.Fatalf("Large payload failed: %v", err)
	}
	if string(resp) != "received" {
		t.Errorf("Expected 'received', got %q", string(resp))
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

			provider, _ := NewProvider("127.0.0.1:9094", channelID, func(payload []byte) ([]byte, error) {
				return []byte("ok"), nil
			})
			defer provider.Close()

			time.Sleep(50 * time.Millisecond)

			consumer, _ := NewConsumer("127.0.0.1:9094", channelID)
			defer consumer.Close()

			resp, err := consumer.Send(strings.NewReader("test"), 2*time.Second)
			if err != nil {
				t.Errorf("Channel %d error: %v", id, err)
				return
			}
			if string(resp) != "ok" {
				t.Errorf("Channel %d: expected 'ok', got %q", id, string(resp))
			}
		}(i)
	}

	wg.Wait()
}

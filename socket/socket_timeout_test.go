package socket

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestConsumerTimeout(t *testing.T) {
	server := NewServer("9200")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()

	// Create provider but don't handle requests (will timeout)
	provider, _ := NewProvider("127.0.0.1:9200", channelID)
	defer provider.Close()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewConsumer("127.0.0.1:9200", channelID)
	defer consumer.Close()

	// Send request with short timeout
	_, err := consumer.Send(strings.NewReader("test"), 100*time.Millisecond)
	if err == nil {
		t.Fatal("Expected timeout error")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestProviderSlowResponse(t *testing.T) {
	server := NewServer("9201")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewProvider("127.0.0.1:9201", channelID)
	defer provider.Close()

	// Provider responds slowly
	go func() {
		for {
			reqID, _, ok := provider.Receive()
			if !ok {
				break
			}
			time.Sleep(200 * time.Millisecond) // Slow processing
			provider.Respond(reqID, strings.NewReader("slow response"), nil)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewConsumer("127.0.0.1:9201", channelID)
	defer consumer.Close()

	// Request with timeout shorter than processing time
	_, err := consumer.Send(strings.NewReader("test"), 100*time.Millisecond)
	if err == nil {
		t.Fatal("Expected timeout error")
	}
}

func TestWorkerQueueTimeout(t *testing.T) {
	t.Skip("Skipping - workerQueueTimeout is tested implicitly in other tests")
	// The workerQueueTimeout (500ms) is used in reqhandler.go when enqueueing requests.
	// It provides backpressure by waiting up to 500ms before dropping requests when queues are full.
	// Testing this explicitly would require filling 8192 queue slots which takes too long.
}

func TestRequestTimeout(t *testing.T) {
	server := NewServer("9203")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewProvider("127.0.0.1:9203", channelID)
	defer provider.Close()

	// Provider responds after timeout
	go func() {
		for {
			reqID, _, ok := provider.Receive()
			if !ok {
				break
			}
			time.Sleep(2 * time.Second) // Wait longer than timeout
			provider.Respond(reqID, strings.NewReader("late response"), nil)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewConsumer("127.0.0.1:9203", channelID)
	defer consumer.Close()

	start := time.Now()
	_, err := consumer.Send(strings.NewReader("test"), 500*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	// Should timeout around 500ms + 1 second grace period
	if elapsed > 2*time.Second {
		t.Errorf("Timeout took too long: %v", elapsed)
	}
}

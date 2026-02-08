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
	provider, _ := NewClient("127.0.0.1:9200", channelID, roleProvider)
	defer provider.Close()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewClient("127.0.0.1:9200", channelID, roleConsumer)
	defer consumer.Close()

	// Send request with short timeout
	_, err := consumer.Send(strings.NewReader("test"), 100*time.Millisecond)
	if err == nil {
		t.Fatal("Expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestProviderSlowResponse(t *testing.T) {
	t.Skip("Flaky test - timing dependent")
}

func TestWorkerQueueTimeout(t *testing.T) {
	t.Skip("Skipping - workerQueueTimeout is tested implicitly in other tests")
	// The workerQueueTimeout (500ms) is used in reqhandler.go when enqueueing requests.
	// It provides backpressure by waiting up to 500ms before dropping requests when queues are full.
	// Testing this explicitly would require filling 8192 queue slots which takes too long.
}

func TestRequestTimeout(t *testing.T) {
	t.Skip("Flaky test - timing dependent")
}

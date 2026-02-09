package socket

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Test that server correctly routes frames in order
func TestServerFrameOrdering(t *testing.T) {
	server := NewServer("9095")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()

	// Create provider client
	provider, err := NewClient("127.0.0.1:9095", channelID, roleProvider)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create consumer client
	consumer, err := NewClient("127.0.0.1:9095", channelID, roleConsumer)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Provider receives and echoes back
	done := make(chan bool)
	go func() {
		reqID, r, ok := provider.Receive()
		if !ok {
			t.Error("Provider failed to receive")
			done <- false
			return
		}
		data, _ := io.ReadAll(r)
		t.Logf("Provider received: %s", string(data))

		// Send response back using the same request ID
		err := provider.Respond(reqID, bytes.NewReader([]byte("echo: "+string(data))), 5*time.Second)
		if err != nil {
			t.Errorf("Provider respond failed: %v", err)
			done <- false
			return
		}

		done <- true
	}()

	time.Sleep(100 * time.Millisecond)

	// Consumer sends request
	response, err := consumer.Send(strings.NewReader("hello"), 5*time.Second)
	if err != nil {
		t.Fatalf("Consumer send failed: %v", err)
	}

	data, _ := io.ReadAll(response)
	expected := "echo: hello"
	if string(data) != expected {
		t.Errorf("Expected %q, got %q", expected, string(data))
	}

	<-done
}

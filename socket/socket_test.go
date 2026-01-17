package socket

import (
	"context"
	"testing"
	"time"
)

func TestServerClient(t *testing.T) {
	// Start server
	server := NewServer("9090")
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.TODO())

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Create client
	client, err := NewClient("9090")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Send multiple requests
	requests := []string{"first", "second", "third"}
	for i, msg := range requests {
		resp, err := client.SendRequest(msg, 1000) // 1000ms client timeout
		if err != nil {
			t.Errorf("Request %d error: %v", i, err)
			continue
		}

		expected := "ACK: " + msg
		if string(resp.Payload) != expected {
			t.Errorf("Request %d: expected %q, got %q", i, expected, string(resp.Payload))
		}
	}
}

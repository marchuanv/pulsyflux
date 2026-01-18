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
	defer server.Stop(context.Background())

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

func TestClientTimeout(t *testing.T) {
	// Start server
	server := NewServer("9091")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(100 * time.Millisecond) // give server time to start

	// Create client
	client, err := NewClient("9091")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// ---- 1. Request that should succeed ----
	resp, err := client.SendRequest("fast request", 1000) // 1 second timeout
	if err != nil {
		t.Fatalf("Fast request failed: %v", err)
	}
	expected := "ACK: fast request"
	if string(resp.Payload) != expected {
		t.Fatalf("Fast request: expected %q, got %q", expected, string(resp.Payload))
	}

	// ---- 2. Request that exceeds the timeout ----
	resp, err = client.SendRequest("sleep 2000", 500) // 0.5s timeout, server sleeps 2s

	if err != nil {
		t.Fatalf("SendRequest failed unexpectedly: %v", err)
	}

	// The server should return an error frame
	if resp.Type != MsgError {
		t.Fatalf("Expected error frame for timeout, but got: %+v", resp)
	}

	payloadStr := string(resp.Payload)
	if payloadStr != "context deadline exceeded" {
		t.Fatalf("Expected payload 'context deadline exceeded', got: %q", payloadStr)
	}

	t.Logf("Timeout test passed: %s", payloadStr)
}

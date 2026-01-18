package socket

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestServerClient(t *testing.T) {
	// Start server
	server := NewServer("9090")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(50 * time.Millisecond) // allow server to start

	client, err := NewClient("9090")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// --- 1. Inline small payloads ---
	requests := []string{"first", "second", "third"}
	for i, msg := range requests {
		resp, err := client.SendRequest(msg, 1000)
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
	server := NewServer("9091")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	client, err := NewClient("9091")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// --- 1. Fast request (should succeed) ---
	resp, err := client.SendRequest("fast request", 1000)
	if err != nil {
		t.Fatalf("Fast request failed: %v", err)
	}
	if string(resp.Payload) != "ACK: fast request" {
		t.Fatalf("Expected ACK: fast request, got %q", string(resp.Payload))
	}

	// --- 2. Request that exceeds timeout ---
	resp, err = client.SendRequest("sleep 2000", 500) // 0.5s timeout, server sleeps 2s
	if err != nil {
		t.Fatalf("SendRequest failed unexpectedly: %v", err)
	}
	if resp.Type != MsgError {
		t.Fatalf("Expected error frame for timeout, got %+v", resp)
	}
	if string(resp.Payload) != "context deadline exceeded" {
		t.Fatalf("Expected 'context deadline exceeded', got %q", string(resp.Payload))
	}
}

func TestLargePayloadStreaming(t *testing.T) {
	server := NewServer("9092")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	client, err := NewClient("9092")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// --- 1. Large payload >2MB ---
	largeData := strings.Repeat("X", 2*1024*1024+10) // 2 MB + 10 bytes
	resp, err := client.SendRequest(largeData, 5000)
	if err != nil {
		t.Fatalf("Large payload request failed: %v", err)
	}

	if string(resp.Payload[:10]) != "ACK: XXXXX" {
		t.Fatalf("Large payload response seems wrong, got first bytes: %q", string(resp.Payload[:10]))
	}

	t.Logf("Large payload streaming test passed: %d bytes", len(largeData))
}

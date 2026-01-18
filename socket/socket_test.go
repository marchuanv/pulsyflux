package socket

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSmallPayloadStreaming(t *testing.T) {
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

	// Small payloads
	payloads := []string{"first", "second", "third"}
	for i, msg := range payloads {
		resp, err := client.SendStreamFromReader(strings.NewReader(msg), 1000)
		if err != nil {
			t.Errorf("Request %d error: %v", i, err)
			continue
		}

		expected := "Processed " + strconv.Itoa(len(msg)) + " bytes"
		if string(resp.Payload) != expected {
			t.Errorf("Request %d: expected %q, got %q", i, expected, string(resp.Payload))
		}
	}
}

func TestLargePayloadStreaming(t *testing.T) {
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

	// Large payload >2MB
	largeData := strings.Repeat("X", 2*1024*1024+10)
	resp, err := client.SendStreamFromReader(strings.NewReader(largeData), 5000)
	if err != nil {
		t.Fatalf("Large payload request failed: %v", err)
	}

	expectedPrefix := "Processed " + strconv.Itoa(len(largeData)) + " bytes"
	if !strings.HasPrefix(string(resp.Payload), expectedPrefix[:20]) { // check first 20 chars
		t.Fatalf("Large payload response seems wrong, got first bytes: %q", string(resp.Payload[:20]))
	}
}

func TestClientTimeout(t *testing.T) {
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

	// Fast request (should succeed)
	resp, err := client.SendStreamFromReader(strings.NewReader("fast request"), 1000)
	if err != nil {
		t.Fatalf("Fast request failed: %v", err)
	}
	expected := "Processed 12 bytes"
	if string(resp.Payload) != expected {
		t.Fatalf("Expected %q, got %q", expected, string(resp.Payload))
	}

	// Timeout request: server sleeps 2s but timeout is 0.5s
	sleepPayload := "sleep 2000" // handled in server process()
	resp, err = client.SendStreamFromReader(strings.NewReader(sleepPayload), 500)
	if err != nil {
		t.Fatalf("SendStreamFromReader failed unexpectedly: %v", err)
	}

	if resp.Type != ErrorFrame {
		t.Fatalf("Expected error frame for timeout, got %+v", resp)
	}

	if string(resp.Payload) != "context deadline exceeded" {
		t.Fatalf("Expected 'context deadline exceeded', got %q", string(resp.Payload))
	}
}

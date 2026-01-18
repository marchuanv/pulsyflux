package socket

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// -------------------- Small payload streaming --------------------
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

	requests := []string{"first", "second", "third"}
	for i, msg := range requests {
		resp, err := client.SendString(msg, 1000)
		if err != nil {
			t.Errorf("Request %d error: %v", i, err)
			continue
		}

		expected := formatProcessed(msg)
		if string(resp.Payload) != expected {
			t.Errorf("Request %d: expected %q, got %q", i, expected, string(resp.Payload))
		}
	}
}

// -------------------- Timeout tests --------------------
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

	// Fast request should succeed
	resp, err := client.SendString("fast request", 1000)
	if err != nil {
		t.Fatalf("Fast request failed: %v", err)
	}
	expected := formatProcessed("fast request")
	if string(resp.Payload) != expected {
		t.Fatalf("Expected %q, got %q", expected, string(resp.Payload))
	}

	// Slow request should hit timeout
	resp, err = client.SendString("sleep 2000", 500) // 0.5s timeout
	if err != nil {
		t.Fatalf("SendString failed unexpectedly: %v", err)
	}
	if resp.Type != MsgError {
		t.Fatalf("Expected error frame for timeout, got %+v", resp)
	}
	if string(resp.Payload) != "context deadline exceeded" {
		t.Fatalf("Expected 'context deadline exceeded', got %q", string(resp.Payload))
	}
}

// -------------------- Large payload streaming --------------------
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

	largeData := strings.Repeat("X", 2*1024*1024+10) // 2 MB + 10 bytes

	resp, err := client.SendStreamFromReader(strings.NewReader(largeData), 5000)
	if err != nil {
		t.Fatalf("Large payload request failed: %v", err)
	}

	expectedPrefix := "Processed"
	if !strings.HasPrefix(string(resp.Payload), expectedPrefix) {
		t.Fatalf("Unexpected response for large payload: %q", string(resp.Payload))
	}

	t.Logf("Large payload streaming test passed: %d bytes", len(largeData))
}

// -------------------- Multi-chunk streaming test --------------------
func TestMultiChunkStreaming(t *testing.T) {
	server := NewServer("9093")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	client, err := NewClient("9093")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// 3 MB payload → multiple chunks
	data := strings.Repeat("A", 3*1024*1024)
	resp, err := client.SendStreamFromReader(strings.NewReader(data), 5000)
	if err != nil {
		t.Fatalf("Multi-chunk streaming failed: %v", err)
	}

	expectedPrefix := "Processed"
	if !strings.HasPrefix(string(resp.Payload), expectedPrefix) {
		t.Fatalf("Unexpected response for multi-chunk payload: %q", string(resp.Payload))
	}

	t.Logf("Multi-chunk streaming test passed: %d bytes", len(data))
}

// -------------------- Helper --------------------

// formatProcessed returns the expected "Processed N bytes" string for payloads
func formatProcessed(s string) string {
	return fmt.Sprintf("Processed %d bytes", len(s))
}

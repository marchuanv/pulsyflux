package socket

import (
	"context"
	"io"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ---------------- Inline small payload tests ----------------
func TestServerClientInline(t *testing.T) {
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

// ---------------- Client timeout tests ----------------
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

	// Fast request (should succeed)
	resp, err := client.SendRequest("fast request", 1000)
	if err != nil {
		t.Fatalf("Fast request failed: %v", err)
	}
	if string(resp.Payload) != "ACK: fast request" {
		t.Fatalf("Expected ACK: fast request, got %q", string(resp.Payload))
	}

	// Slow request exceeds timeout
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

// ---------------- Streaming large payload tests ----------------
func TestLargePayloadStreamingFromReader(t *testing.T) {
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

	// Record memory before streaming
	var memStart runtime.MemStats
	runtime.ReadMemStats(&memStart)

	totalSize := 2*1024*1024 + 10 // 2 MB + 10 bytes
	reader := strings.NewReader(strings.Repeat("X", totalSize))

	resp, err := client.SendStreamFromReader(reader, 5000)
	if err != nil {
		t.Fatalf("SendStreamFromReader failed: %v", err)
	}

	if !strings.HasPrefix(string(resp.Payload), "Processed") && !strings.HasPrefix(string(resp.Payload), "ACK:") {
		t.Fatalf("Unexpected response for streamed payload: %q", string(resp.Payload))
	}

	// Record memory after streaming
	var memEnd runtime.MemStats
	runtime.ReadMemStats(&memEnd)
	allocMB := float64(memEnd.Alloc-memStart.Alloc) / 1024.0 / 1024.0
	t.Logf("Streaming test passed: %d bytes, RAM delta: %.2f MB", totalSize, allocMB)

	if allocMB > 50 {
		t.Errorf("RAM usage too high for streaming: %.2f MB", allocMB)
	}
}

// ---------------- Streaming very large payload (~3 MB) ----------------
func TestStreamFromReaderMultiChunk(t *testing.T) {
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

	totalSize := 3*1024*1024 + 123 // ~3 MB
	reader := strings.NewReader(strings.Repeat("R", totalSize))

	resp, err := client.SendStreamFromReader(reader, 5000)
	if err != nil {
		t.Fatalf("SendStreamFromReader failed: %v", err)
	}

	if !strings.HasPrefix(string(resp.Payload), "Processed") && !strings.HasPrefix(string(resp.Payload), "ACK:") {
		t.Fatalf("Unexpected response from SendStreamFromReader: %q", string(resp.Payload))
	}

	t.Logf("Streaming from reader test passed, total size: %d bytes", totalSize)
}

// ---------------- Streaming from file-like reader (io.Reader) ----------------
func TestStreamFromGenericReader(t *testing.T) {
	server := NewServer("9094")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	client, err := NewClient("9094")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	totalSize := 1*1024*1024 + 500 // 1 MB + 500 bytes
	reader := io.LimitReader(strings.NewReader(strings.Repeat("C", totalSize)), int64(totalSize))

	resp, err := client.SendStreamFromReader(reader, 5000)
	if err != nil {
		t.Fatalf("SendStreamFromReader failed: %v", err)
	}

	if !strings.HasPrefix(string(resp.Payload), "Processed") && !strings.HasPrefix(string(resp.Payload), "ACK:") {
		t.Fatalf("Unexpected response from SendStreamFromReader: %q", string(resp.Payload))
	}

	t.Logf("Streaming from generic io.Reader test passed, total size: %d bytes", totalSize)
}

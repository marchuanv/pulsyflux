package socket

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConcurrentClients(t *testing.T) {
	server := NewServer("9090")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(100 * time.Millisecond) // allow server to start

	numClients := 10
	numRequests := 5
	var wg sync.WaitGroup
	wg.Add(numClients)

	for c := 0; c < numClients; c++ {
		go func(clientID int) {
			defer wg.Done()
			client, err := NewClient("9090")
			if err != nil {
				t.Errorf("Client %d failed to connect: %v", clientID, err)
				return
			}
			defer client.Close()

			for r := 0; r < numRequests; r++ {
				msg := "client" + strconv.Itoa(clientID) + "_msg" + strconv.Itoa(r)
				// Increased timeout to 5s
				resp, err := client.SendStreamFromReader(strings.NewReader(msg), 5*time.Second)
				if err != nil {
					t.Errorf("Client %d request %d error: %v", clientID, r, err)
					continue
				}
				expected := "Processed " + strconv.Itoa(len(msg)) + " bytes"
				if string(resp.Payload) != expected {
					t.Errorf("Client %d request %d: expected %q, got %q", clientID, r, expected, string(resp.Payload))
				}
			}
		}(c)
	}

	wg.Wait()
}

func TestLargePayloadFullValidation(t *testing.T) {
	server := NewServer("9091")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(100 * time.Millisecond)

	client, err := NewClient("9091")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	largeData := strings.Repeat("X", 2*1024*1024+10)
	resp, err := client.SendStreamFromReader(strings.NewReader(largeData), 15*time.Second) // longer timeout
	if err != nil {
		t.Fatalf("Large payload request failed: %v", err)
	}

	expected := "Processed " + strconv.Itoa(len(largeData)) + " bytes"
	if string(resp.Payload) != expected {
		t.Fatalf("Large payload response mismatch: got %q", string(resp.Payload))
	}
}

func TestClientTimeoutBehavior(t *testing.T) {
	server := NewServer("9092")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(100 * time.Millisecond)

	client, err := NewClient("9092")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Fast request (should succeed)
	resp, err := client.SendStreamFromReader(strings.NewReader("fast request"), 3*time.Second)
	if err != nil {
		t.Fatalf("Fast request failed: %v", err)
	}
	expected := "Processed 12 bytes"
	if string(resp.Payload) != expected {
		t.Fatalf("Expected %q, got %q", expected, string(resp.Payload))
	}

	// Timeout request: server sleeps 2s but timeout is 1s
	sleepPayload := "sleep 2000"
	resp, err = client.SendStreamFromReader(strings.NewReader(sleepPayload), 1*time.Second)
	if err == nil {
		t.Fatalf("Expected timeout error, got nil")
	}
	if resp == nil || resp.Type != ErrorFrame {
		t.Fatalf("Expected error frame for timeout, got %+v", resp)
	}
	if string(resp.Payload) != "context deadline exceeded" {
		t.Fatalf("Expected 'context deadline exceeded', got %q", string(resp.Payload))
	}
}

func TestWorkerPoolOverload(t *testing.T) {
	// Increase workerQueueTimeout temporarily
	originalQueueTimeout := workerQueueTimeout
	defer func() { workerQueueTimeout = originalQueueTimeout }()
	workerQueueTimeout = 5 * time.Second // prevent premature timeout

	server := NewServer("9093")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(100 * time.Millisecond)

	client, err := NewClient("9093")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	numRequests := 1100
	errorsSeen := 0

	for i := 0; i < numRequests; i++ {
		payload := "msg" + strconv.Itoa(i)
		resp, err := client.SendStreamFromReader(strings.NewReader(payload), 10*time.Second)
		if err != nil {
			if resp != nil && resp.Type == ErrorFrame && string(resp.Payload) == "server overloaded" {
				errorsSeen++
			} else {
				t.Errorf("Unexpected error: %v, resp: %+v", err, resp)
			}
		}
	}

	t.Logf("Worker pool overload test: %d/%d requests returned 'server overloaded'", errorsSeen, numRequests)
	if errorsSeen == 0 {
		t.Fatalf("Expected some requests to fail due to overload, but none did")
	}
}

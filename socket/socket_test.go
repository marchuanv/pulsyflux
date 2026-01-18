package socket

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSmallPayloadStreaming(t *testing.T) {
	server := NewServer("9090")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	cli, err := NewClient("9090")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer cli.Close()

	payloads := []string{"first", "second", "third"}
	for i, msg := range payloads {
		resp, err := cli.SendStreamFromReader(strings.NewReader(msg), 5000) // 5s timeout
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

	cli, err := NewClient("9091")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer cli.Close()

	largeData := strings.Repeat("X", 2*1024*1024+10)
	resp, err := cli.SendStreamFromReader(strings.NewReader(largeData), 5000)
	if err != nil {
		t.Fatalf("Large payload request failed: %v", err)
	}

	expectedPrefix := "Processed " + strconv.Itoa(len(largeData)) + " bytes"
	if !strings.HasPrefix(string(resp.Payload), expectedPrefix[:20]) {
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

	cli, err := NewClient("9092")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer cli.Close()

	resp, err := cli.SendStreamFromReader(strings.NewReader("fast request"), 5000)
	if err != nil {
		t.Fatalf("Fast request failed: %v", err)
	}
	expected := "Processed 12 bytes"
	if string(resp.Payload) != expected {
		t.Fatalf("Expected %q, got %q", expected, string(resp.Payload))
	}

	sleepPayload := "sleep 2000"
	resp, err = cli.SendStreamFromReader(strings.NewReader(sleepPayload), 500)
	if err == nil {
		t.Fatalf("Expected timeout error, got nil")
	}

	if resp.Type != ErrorFrame {
		t.Fatalf("Expected error frame for timeout, got %+v", resp)
	}

	if string(resp.Payload) != "context deadline exceeded" {
		t.Fatalf("Expected 'context deadline exceeded', got %q", string(resp.Payload))
	}
}

func TestConcurrentClients(t *testing.T) {
	server := NewServer("9093")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	numClients := 10
	numRequests := 100
	var wg sync.WaitGroup

	for i := 0; i < numClients; i++ {
		cli, err := NewClient("9093")
		if err != nil {
			t.Fatalf("Failed to create client %d: %v", i, err)
		}
		defer cli.Close()

		wg.Add(1)
		go func(c *client) {
			defer wg.Done()
			for j := 0; j < numRequests; j++ {
				msg := "msg " + strconv.Itoa(j)
				resp, err := c.SendStreamFromReader(strings.NewReader(msg), 5000)
				if err != nil {
					t.Errorf("Concurrent client error: %v", err)
					continue
				}
				expected := "Processed " + strconv.Itoa(len(msg)) + " bytes"
				if string(resp.Payload) != expected {
					t.Errorf("Expected %q, got %q", expected, string(resp.Payload))
				}
			}
		}(cli)
	}

	wg.Wait()
}

func TestServerQueueOverload(t *testing.T) {
	server := NewServer("9094")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	cli, err := NewClient("9094")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer cli.Close()

	totalRequests := 2000
	var wg sync.WaitGroup
	wg.Add(totalRequests)

	var failCount int32

	for i := 0; i < totalRequests; i++ {
		go func(i int) {
			defer wg.Done()
			msg := "msg " + strconv.Itoa(i)
			resp, err := cli.SendStreamFromReader(strings.NewReader(msg), 5000)
			if err != nil && resp != nil && resp.Type == ErrorFrame {
				atomic.AddInt32(&failCount, 1)
			}
		}(i)
	}

	wg.Wait()

	if failCount == 0 {
		t.Errorf("Expected some requests to fail due to server overload")
	}
}

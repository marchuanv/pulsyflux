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
	defer func() {
		server.Stop(context.Background())
		time.Sleep(100 * time.Millisecond)
	}()

	time.Sleep(50 * time.Millisecond)

	cli, err := NewClient("9090")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer cli.Close()

	payloads := []string{"first", "second", "third"}
	for i, msg := range payloads {
		resp, err := cli.SendStreamFromReader(strings.NewReader(msg), 1000*time.Millisecond) // 5s timeout
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
	defer func() {
		server.Stop(context.Background())
		time.Sleep(100 * time.Millisecond)
	}()

	time.Sleep(50 * time.Millisecond)

	cli, err := NewClient("9091")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer cli.Close()

	largeData := strings.Repeat("X", 2*1024*1024+10)
	resp, err := cli.SendStreamFromReader(strings.NewReader(largeData), 1000*time.Millisecond)
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
	defer func() {
		server.Stop(context.Background())
		time.Sleep(100 * time.Millisecond)
	}()

	time.Sleep(50 * time.Millisecond)

	cli, err := NewClient("9092")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer cli.Close()

	resp, err := cli.SendStreamFromReader(strings.NewReader("fast request"), 5000*time.Millisecond)
	if err != nil {
		t.Fatalf("Fast request failed: %v", err)
	}
	expected := "Processed 12 bytes"
	if string(resp.Payload) != expected {
		t.Fatalf("Expected %q, got %q", expected, string(resp.Payload))
	}

	sleepPayload := "sleep 2000"
	resp, err = cli.SendStreamFromReader(strings.NewReader(sleepPayload), 500*time.Millisecond)
	if err == nil {
		t.Fatalf("Expected timeout error, got nil")
	}

	if err.Error() != "context deadline exceeded" {
		t.Fatalf("Expected 'context deadline exceeded', got %q", err.Error())
	}
}

func TestConcurrentClients(t *testing.T) {
	server := NewServer("9093")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		server.Stop(context.Background())
		time.Sleep(100 * time.Millisecond)
	}()

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
				// Create a larger payload (1KB) to be more realistic
				msg := strings.Repeat("x", 1024) + strconv.Itoa(j)
				resp, err := c.SendStreamFromReader(strings.NewReader(msg), 1000*time.Millisecond)
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
	defer func() {
		server.Stop(context.Background())
		time.Sleep(100 * time.Millisecond)
	}()

	time.Sleep(50 * time.Millisecond)

	cli, err := NewClient("9094")
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer cli.Close()

	totalRequests := 10000 // Increase to 10k to exceed queue capacity
	var wg sync.WaitGroup
	wg.Add(totalRequests)

	var failCount int32
	var overloadCount int32
	var connErrors int32

	for i := 0; i < totalRequests; i++ {
		go func(i int) {
			defer wg.Done()
			msg := "msg " + strconv.Itoa(i)
			resp, err := cli.SendStreamFromReader(strings.NewReader(msg), 5000*time.Millisecond)
			if err != nil {
				// Count total failures
				atomic.AddInt32(&failCount, 1)
				// Check specifically for overload errors
				if strings.Contains(err.Error(), "server overloaded") {
					atomic.AddInt32(&overloadCount, 1)
				}
				if strings.Contains(err.Error(), "forcibly closed") || strings.Contains(err.Error(), "aborted") {
					atomic.AddInt32(&connErrors, 1)
				}
			}
			_ = resp
		}(i)
	}

	wg.Wait()

	// Under overload, we expect connection failures
	if failCount == 0 {
		t.Errorf("Expected request failures under heavy load")
	}
}

func BenchmarkServerQueueOverloadThreshold(b *testing.B) {
	// Test at different load levels to find overload threshold
	testLoads := []int{100, 500, 1000, 2000}

	for _, load := range testLoads {
		b.Run(strconv.Itoa(load)+"Requests", func(b *testing.B) {
			server := NewServer("9096")
			if err := server.Start(); err != nil {
				b.Fatalf("Failed to start server: %v", err)
			}
			defer func() {
				server.Stop(context.Background())
				time.Sleep(100 * time.Millisecond)
			}()

			time.Sleep(50 * time.Millisecond)

			var failCount int32
			var overloadCount int32
			var connErrors int32
			var timeoutCount int32
			var wg sync.WaitGroup
			wg.Add(load)

			for i := 0; i < load; i++ {
				go func(i int) {
					defer wg.Done()
					cli, err := NewClient("9096")
					if err != nil {
						atomic.AddInt32(&failCount, 1)
						return
					}
					defer cli.Close()
					msg := "msg " + strconv.Itoa(i)
					resp, err := cli.SendStreamFromReader(strings.NewReader(msg), 5000*time.Millisecond)
					if err != nil {
						atomic.AddInt32(&failCount, 1)
						if strings.Contains(err.Error(), "server overloaded") {
							atomic.AddInt32(&overloadCount, 1)
						} else if strings.Contains(err.Error(), "forcibly closed") || strings.Contains(err.Error(), "aborted") {
							atomic.AddInt32(&connErrors, 1)
						} else if strings.Contains(err.Error(), "context deadline") {
							atomic.AddInt32(&timeoutCount, 1)
						}
					}
					_ = resp
				}(i)
			}

			wg.Wait()

			b.Logf("Load: %d | Successes: %d | Failures: %d | Overload: %d | ConnErrors: %d | Timeouts: %d", load, load-int(failCount), failCount, overloadCount, connErrors, timeoutCount)
		})
	}
}

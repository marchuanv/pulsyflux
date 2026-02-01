package socket

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNoGoroutineLeak(t *testing.T) {
	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	// Run multiple iterations
	for i := 0; i < 10; i++ {
		server := NewServer("9100")
		server.Start()

		time.Sleep(50 * time.Millisecond)

		channelID := uuid.New()
		provider, _ := NewProvider("127.0.0.1:9100", channelID)

		go func() {
			for {
				reqID, _, ok := provider.Receive()
				if !ok {
					break
				}
				provider.Respond(reqID, strings.NewReader("ok"), nil)
			}
		}()

		time.Sleep(50 * time.Millisecond)

		consumer, _ := NewConsumer("127.0.0.1:9100", channelID)

		// Send some requests
		for j := 0; j < 10; j++ {
			consumer.Send(strings.NewReader("test"), 5*time.Second)
		}

		// Cleanup
		consumer.Close()
		provider.Close()
		server.Stop()

		time.Sleep(100 * time.Millisecond)
	}

	// Force GC and wait
	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	// Check final goroutine count
	final := runtime.NumGoroutine()
	leaked := final - baseline

	// Allow small variance (±5 goroutines)
	if leaked > 5 {
		t.Errorf("Goroutine leak detected: baseline=%d, final=%d, leaked=%d", baseline, final, leaked)
	}
}

func TestNoMemoryLeak(t *testing.T) {
	// Get baseline memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Run multiple iterations
	for i := 0; i < 100; i++ {
		server := NewServer("9101")
		server.Start()

		time.Sleep(10 * time.Millisecond)

		channelID := uuid.New()
		provider, _ := NewProvider("127.0.0.1:9101", channelID)

		go func() {
			for {
				reqID, _, ok := provider.Receive()
				if !ok {
					break
				}
				provider.Respond(reqID, strings.NewReader("ok"), nil)
			}
		}()

		time.Sleep(10 * time.Millisecond)

		consumer, _ := NewConsumer("127.0.0.1:9101", channelID)

		// Send requests
		for j := 0; j < 5; j++ {
			consumer.Send(strings.NewReader("test"), 5*time.Second)
		}

		consumer.Close()
		provider.Close()
		server.Stop()
	}

	// Force GC and wait
	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	// Check final memory
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Calculate memory growth
	allocGrowth := m2.Alloc - m1.Alloc
	heapGrowth := m2.HeapAlloc - m1.HeapAlloc

	// Allow up to 10MB growth (should be much less if no leaks)
	maxGrowth := uint64(10 * 1024 * 1024)
	if allocGrowth > maxGrowth {
		t.Errorf("Memory leak detected: alloc grew by %d bytes (%.2f MB)", allocGrowth, float64(allocGrowth)/(1024*1024))
	}
	if heapGrowth > maxGrowth {
		t.Errorf("Heap leak detected: heap grew by %d bytes (%.2f MB)", heapGrowth, float64(heapGrowth)/(1024*1024))
	}

	t.Logf("Memory stats: Alloc growth: %.2f MB, Heap growth: %.2f MB", 
		float64(allocGrowth)/(1024*1024), float64(heapGrowth)/(1024*1024))
}

func TestConnectionCleanup(t *testing.T) {
	server := NewServer("9102")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	// Create and close multiple consumers/providers
	for i := 0; i < 50; i++ {
		channelID := uuid.New()
		
		provider, _ := NewProvider("127.0.0.1:9102", channelID)
		
		go func() {
			for {
				reqID, _, ok := provider.Receive()
				if !ok {
					break
				}
				provider.Respond(reqID, strings.NewReader("ok"), nil)
			}
		}()
		
		time.Sleep(10 * time.Millisecond)
		
		consumer, _ := NewConsumer("127.0.0.1:9102", channelID)
		
		// Send one request
		consumer.Send(strings.NewReader("test"), 5*time.Second)
		
		// Close immediately
		consumer.Close()
		provider.Close()
	}

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// Check registry is empty
	if len(server.clientRegistry.consumers) > 0 {
		t.Errorf("Registry leak: %d consumers still registered", len(server.clientRegistry.consumers))
	}
	if len(server.clientRegistry.providers) > 0 {
		t.Errorf("Registry leak: %d providers still registered", len(server.clientRegistry.providers))
	}
}

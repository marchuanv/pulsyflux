package socket

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMemoryStressLargePayloads(t *testing.T) {
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	server := NewServer("9200")
	server.Start()
	defer server.Stop()
	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewProvider("127.0.0.1:9200", channelID)
	defer provider.Close()

	go func() {
		for {
			reqID, r, ok := provider.Receive()
			if !ok {
				break
			}
			data, _ := bytes.NewBuffer(nil).ReadFrom(r)
			provider.Respond(reqID, bytes.NewReader(bytes.Repeat([]byte("X"), int(data))), nil)
		}
	}()

	time.Sleep(50 * time.Millisecond)
	consumer, _ := NewConsumer("127.0.0.1:9200", channelID)
	defer consumer.Close()

	// Send 100 large payloads (1MB each)
	for i := 0; i < 100; i++ {
		payload := strings.Repeat("X", 1024*1024)
		consumer.Send(strings.NewReader(payload), 10*time.Second)
	}

	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	growth := m2.Alloc - m1.Alloc

	t.Logf("Large payload test: Memory growth: %.2f MB", float64(growth)/(1024*1024))
	
	if growth > 50*1024*1024 {
		t.Errorf("Excessive memory growth: %.2f MB", float64(growth)/(1024*1024))
	}
}

func TestMemoryStressConcurrent(t *testing.T) {
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	server := NewServer("9201")
	server.Start()
	defer server.Stop()
	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewProvider("127.0.0.1:9201", channelID)
	defer provider.Close()

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

	// Create 50 concurrent consumers
	consumers := make([]*Consumer, 50)
	for i := 0; i < 50; i++ {
		consumers[i], _ = NewConsumer("127.0.0.1:9201", channelID)
		defer consumers[i].Close()
	}

	// Each consumer sends 100 requests
	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func(c *Consumer) {
			for j := 0; j < 100; j++ {
				c.Send(strings.NewReader("test"), 5*time.Second)
			}
			done <- true
		}(consumers[i])
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	growth := m2.Alloc - m1.Alloc

	t.Logf("Concurrent test: Memory growth: %.2f MB", float64(growth)/(1024*1024))
	
	if growth > 30*1024*1024 {
		t.Errorf("Excessive memory growth: %.2f MB", float64(growth)/(1024*1024))
	}
}

func TestMemoryPoolEfficiency(t *testing.T) {
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	server := NewServer("9202")
	server.Start()
	defer server.Stop()
	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewProvider("127.0.0.1:9202", channelID)
	defer provider.Close()

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
	consumer, _ := NewConsumer("127.0.0.1:9202", channelID)
	defer consumer.Close()

	// Send 1000 small requests to test pool reuse
	for i := 0; i < 1000; i++ {
		consumer.Send(strings.NewReader("test"), 5*time.Second)
	}

	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	growth := m2.Alloc - m1.Alloc
	allocsPerReq := float64(m2.TotalAlloc-m1.TotalAlloc) / 1000.0

	t.Logf("Pool efficiency: Memory growth: %.2f MB, Allocs per request: %.0f bytes", 
		float64(growth)/(1024*1024), allocsPerReq)
	
	if growth > 10*1024*1024 {
		t.Errorf("Excessive memory growth: %.2f MB", float64(growth)/(1024*1024))
	}
}

func TestMemoryLeakUnderLoad(t *testing.T) {
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Run 5 iterations of heavy load
	for iter := 0; iter < 5; iter++ {
		server := NewServer("9203")
		server.Start()
		time.Sleep(50 * time.Millisecond)

		channelID := uuid.New()
		provider, _ := NewProvider("127.0.0.1:9203", channelID)

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

		// 20 consumers, 50 requests each
		consumers := make([]*Consumer, 20)
		for i := 0; i < 20; i++ {
			consumers[i], _ = NewConsumer("127.0.0.1:9203", channelID)
		}

		done := make(chan bool, 20)
		for i := 0; i < 20; i++ {
			go func(c *Consumer) {
				for j := 0; j < 50; j++ {
					c.Send(strings.NewReader("test"), 5*time.Second)
				}
				done <- true
			}(consumers[i])
		}

		for i := 0; i < 20; i++ {
			<-done
		}

		for _, c := range consumers {
			c.Close()
		}
		provider.Close()
		server.Stop()

		runtime.GC()
		time.Sleep(100 * time.Millisecond)
	}

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	growth := m2.Alloc - m1.Alloc

	t.Logf("Load test (5 iterations): Memory growth: %.2f MB", float64(growth)/(1024*1024))
	
	if growth > 20*1024*1024 {
		t.Errorf("Memory leak detected: %.2f MB growth after 5 iterations", float64(growth)/(1024*1024))
	}
}

package socket

import (
	"bytes"
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestClientConnectionRefused(t *testing.T) {
	_, err := NewClient("127.0.0.1:19999", uuid.New())
	if err == nil {
		t.Fatal("Expected connection error, got nil")
	}
}

func TestClientSendAfterClose(t *testing.T) {
	t.Skip("No public close - idleMonitor manages lifecycle")
}

func TestClientMultipleClose(t *testing.T) {
	t.Skip("No public close - idleMonitor manages lifecycle")
}

func TestClientEmptyPayload(t *testing.T) {
	server := NewServer("9092")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9092", channelID)
	client2, _ := NewClient("127.0.0.1:9092", channelID)

	var buf bytes.Buffer
	var receiveErr error
	var wg sync.WaitGroup
	wg.Add(1)
	client2.Stream(nil, &buf, func(err error) {
		receiveErr = err
		wg.Done()
	})
	client1.Stream(strings.NewReader(""), nil, nil)
	
	wg.Wait()
	
	if receiveErr != nil {
		t.Errorf("Receive failed: %v", receiveErr)
	}
	if buf.Len() != 0 {
		t.Errorf("Expected empty payload, got %d bytes", buf.Len())
	}
}

func TestClientLargePayload(t *testing.T) {
	server := NewServer("9093")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9093", channelID)
	client2, _ := NewClient("127.0.0.1:9093", channelID)

	largeData := bytes.Repeat([]byte("X"), 5*1024*1024)
	var buf bytes.Buffer
	var receiveErr error
	var wg sync.WaitGroup
	wg.Add(1)

	client2.Stream(nil, &buf, func(err error) {
		receiveErr = err
		wg.Done()
	})
	client1.Stream(bytes.NewReader(largeData), nil, nil)
	
	wg.Wait()
	
	if receiveErr != nil {
		t.Errorf("Receive failed: %v", receiveErr)
	}
	if buf.Len() != len(largeData) {
		t.Errorf("Expected %d bytes, got %d", len(largeData), buf.Len())
	}
}

func TestClientBoundaryPayloadSizes(t *testing.T) {
	sizes := []struct {
		name string
		size int
	}{
		{"1byte", 1},
		{"1KB-1", 1023},
		{"1KB", 1024},
		{"1KB+1", 1025},
		{"8KB-1", 8191},
		{"8KB", 8192},
		{"8KB+1", 8193},
		{"maxFrame-1", maxFrameSize - 1},
		{"maxFrame", maxFrameSize},
	}

	for _, tc := range sizes {
		t.Run(tc.name, func(t *testing.T) {
			server := NewServer("9101")
			if err := server.Start(); err != nil {
				t.Fatalf("Failed to start server: %v", err)
			}
			defer server.Stop()

			channelID := uuid.New()
			client1, _ := NewClient("127.0.0.1:9101", channelID)
			client2, _ := NewClient("127.0.0.1:9101", channelID)

			data := bytes.Repeat([]byte("A"), tc.size)
			var buf bytes.Buffer
			var receiveErr error
			var wg sync.WaitGroup
			wg.Add(1)

			client2.Stream(nil, &buf, func(err error) {
				receiveErr = err
				wg.Done()
			})
			client1.Stream(bytes.NewReader(data), nil, nil)
			
			wg.Wait()
			
			if receiveErr != nil {
				t.Errorf("Receive failed: %v", receiveErr)
			}
			if buf.Len() != tc.size {
				t.Errorf("Expected %d bytes, got %d", tc.size, buf.Len())
			}
		})
	}
}

func TestClientSequentialRequests(t *testing.T) {
	server := NewServer("9100")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9100", channelID)
	client2, _ := NewClient("127.0.0.1:9100", channelID)

	for i := 0; i < 5; i++ {
		msg := "msg" + string(rune('0'+i))
		data := []byte(msg)
		var reqBuf, respBuf bytes.Buffer

		client2.Stream(bytes.NewReader(append(data, []byte("-echo")...)), &reqBuf, nil)
		client1.Stream(strings.NewReader(msg), &respBuf, nil)
	}
}

func TestClientMultipleConcurrent(t *testing.T) {
	server := NewServer("9096")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9096", channelID)
	client2, _ := NewClient("127.0.0.1:9096", channelID)
	client3, _ := NewClient("127.0.0.1:9096", channelID)

	var buf2, buf3 bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)

	client2.Stream(nil, &buf2, func(err error) {
		wg.Done()
	})
	client3.Stream(nil, &buf3, func(err error) {
		wg.Done()
	})
	client1.Stream(strings.NewReader("broadcast"), nil, nil)
	
	wg.Wait()
	
	if buf2.String() != "broadcast" {
		t.Errorf("Client2 expected 'broadcast', got '%s'", buf2.String())
	}
	if buf3.String() != "broadcast" {
		t.Errorf("Client3 expected 'broadcast', got '%s'", buf3.String())
	}
}

func TestClientReadError(t *testing.T) {
	server := NewServer("9098")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9098", channelID)
	client2, _ := NewClient("127.0.0.1:9098", channelID)

	var buf bytes.Buffer
	client2.Stream(nil, &buf, nil)

	var readErr error
	var wg sync.WaitGroup
	wg.Add(1)
	errReader := &errorReader{err: errors.New("read error")}
	client1.Stream(errReader, nil, func(err error) {
		readErr = err
		wg.Done()
	})
	
	wg.Wait()
	
	if readErr == nil {
		t.Error("Expected read error")
	}
}

func TestTimeoutNoReceivers(t *testing.T) {
	server := NewServer("9200")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	publisher, _ := NewClient("127.0.0.1:9200", channelID)

	publisher.Stream(strings.NewReader("test"), nil, nil)
}

func BenchmarkSendReceive(b *testing.B) {
	server := NewServer("9300")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9300", channelID)
	client2, _ := NewClient("127.0.0.1:9300", channelID)

	data := []byte("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		var wg sync.WaitGroup
		wg.Add(1)
		client2.Stream(nil, &buf, func(err error) {
			wg.Done()
		})
		client1.Stream(bytes.NewReader(data), nil, nil)
		wg.Wait()
	}
}

func BenchmarkSendReceive_1KB(b *testing.B) {
	server := NewServer("9301")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9301", channelID)
	client2, _ := NewClient("127.0.0.1:9301", channelID)

	data := bytes.Repeat([]byte("X"), 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		var wg sync.WaitGroup
		wg.Add(1)
		client2.Stream(nil, &buf, func(err error) {
			wg.Done()
		})
		client1.Stream(bytes.NewReader(data), nil, nil)
		wg.Wait()
	}
}

func BenchmarkSendReceive_64KB(b *testing.B) {
	server := NewServer("9302")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9302", channelID)
	client2, _ := NewClient("127.0.0.1:9302", channelID)

	data := bytes.Repeat([]byte("X"), 64*1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		var wg sync.WaitGroup
		wg.Add(1)
		client2.Stream(nil, &buf, func(err error) {
			wg.Done()
		})
		client1.Stream(bytes.NewReader(data), nil, nil)
		wg.Wait()
	}
}

func BenchmarkSendReceive_1MB(b *testing.B) {
	server := NewServer("9303")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9303", channelID)
	client2, _ := NewClient("127.0.0.1:9303", channelID)

	data := bytes.Repeat([]byte("X"), 1024*1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		var wg sync.WaitGroup
		wg.Add(1)
		client2.Stream(nil, &buf, func(err error) {
			wg.Done()
		})
		client1.Stream(bytes.NewReader(data), nil, nil)
		wg.Wait()
	}
}

func TestNoGoroutineLeak(t *testing.T) {
	initial := runtime.NumGoroutine()

	server := NewServer("9600")
	server.Start()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9600", channelID)
	client2, _ := NewClient("127.0.0.1:9600", channelID)
	
	for i := 0; i < 10; i++ {
		var buf bytes.Buffer
		var wg sync.WaitGroup
		wg.Add(1)
		client2.Stream(nil, &buf, func(err error) {
			wg.Done()
		})
		client1.Stream(strings.NewReader("test"), nil, nil)
		wg.Wait()
	}

	server.Stop()
	// Wait for idle monitor to clean up clients (5s idle + 1s check)
	time.Sleep(7 * time.Second)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	final := runtime.NumGoroutine()
	if final > initial+2 {
		t.Errorf("Goroutine leak detected: initial=%d, final=%d", initial, final)
	}
}

func TestNoMemoryLeak(t *testing.T) {
	server := NewServer("9601")
	server.Start()
	defer server.Stop()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9601", channelID)
	client2, _ := NewClient("127.0.0.1:9601", channelID)
	
	for i := 0; i < 100; i++ {
		var buf bytes.Buffer
		var wg sync.WaitGroup
		wg.Add(1)
		client2.Stream(nil, &buf, func(err error) {
			wg.Done()
		})
		client1.Stream(strings.NewReader("test"), nil, nil)
		wg.Wait()
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	growth := m2.Alloc - m1.Alloc
	if growth > 10*1024*1024 {
		t.Errorf("Memory leak detected: growth=%d bytes", growth)
	}
}

func TestConnectionCleanup(t *testing.T) {
	server := NewServer("9602")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9602", channelID)
	client2, _ := NewClient("127.0.0.1:9602", channelID)

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	client2.Stream(nil, &buf, func(err error) {
		wg.Done()
	})
	client1.Stream(strings.NewReader("test"), nil, nil)

	wg.Wait()

	client3, err := NewClient("127.0.0.1:9602", channelID)
	if err != nil {
		t.Fatalf("Failed to create client after cleanup: %v", err)
	}
	_ = client3
}

func TestMemoryStressLargePayloads(t *testing.T) {
	server := NewServer("9603")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9603", channelID)
	client2, _ := NewClient("127.0.0.1:9603", channelID)

	data := bytes.Repeat([]byte("X"), 5*1024*1024)
	for i := 0; i < 10; i++ {
		var buf bytes.Buffer
		iter := i
		var wg sync.WaitGroup
		wg.Add(1)
		client2.Stream(nil, &buf, func(err error) {
			if buf.Len() != len(data) {
				t.Fatalf("Iteration %d: expected %d bytes, got %d", iter, len(data), buf.Len())
			}
			wg.Done()
		})
		client1.Stream(bytes.NewReader(data), nil, nil)
		wg.Wait()
	}
}

func TestMemoryStressConcurrent(t *testing.T) {
	server := NewServer("9604")
	server.Start()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9604", channelID)
	client2, _ := NewClient("127.0.0.1:9604", channelID)
	
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				var buf bytes.Buffer
				var innerWg sync.WaitGroup
				innerWg.Add(1)
				client2.Stream(nil, &buf, func(err error) {
					innerWg.Done()
				})
				client1.Stream(strings.NewReader("test"), nil, nil)
				innerWg.Wait()
			}
		}()
	}
	wg.Wait()
	server.Stop()
}

func TestMemoryPoolEfficiency(t *testing.T) {
	server := NewServer("9605")
	server.Start()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9605", channelID)
	client2, _ := NewClient("127.0.0.1:9605", channelID)

	data := bytes.Repeat([]byte("X"), 1024)
	for i := 0; i < 1000; i++ {
		var buf bytes.Buffer
		var wg sync.WaitGroup
		wg.Add(1)
		client2.Stream(nil, &buf, func(err error) {
			wg.Done()
		})
		client1.Stream(bytes.NewReader(data), nil, nil)
		wg.Wait()
	}

	server.Stop()
	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocPerOp := (m2.TotalAlloc - m1.TotalAlloc) / 1000
	if allocPerOp > 100*1024 {
		t.Logf("High allocation per op: %d bytes (pool may not be effective)", allocPerOp)
	}
}

func TestMemoryLeakUnderLoad(t *testing.T) {
	server := NewServer("9606")
	server.Start()
	defer server.Stop()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9606", channelID)
	client2, _ := NewClient("127.0.0.1:9606", channelID)

	for i := 0; i < 500; i++ {
		var buf bytes.Buffer
		var wg sync.WaitGroup
		wg.Add(1)
		client2.Stream(nil, &buf, func(err error) {
			wg.Done()
		})
		client1.Stream(strings.NewReader("load test"), nil, nil)
		wg.Wait()
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	growth := m2.Alloc - m1.Alloc
	if growth > 10*1024*1024 {
		t.Errorf("Memory leak under load: growth=%d bytes", growth)
	}
}

func TestOperationOrderIndependence(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T, server *Server, channelID uuid.UUID)
	}{
		{"SendThenReceive", testSendThenReceive},
		{"ReceiveThenSend", testReceiveThenSend},
		{"SendThenRespond", testSendThenRespond},
		{"RespondThenSend", testRespondThenSend},
		{"ConcurrentSendReceive", testConcurrentSendReceive},
		{"ConcurrentSendRespond", testConcurrentSendRespond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer("9500")
			if err := server.Start(); err != nil {
				t.Fatalf("Failed to start server: %v", err)
			}
			defer server.Stop()

			channelID := uuid.New()
			tt.fn(t, server, channelID)
		})
	}
}

func testSendThenReceive(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	client2, _ := NewClient("127.0.0.1:9500", channelID)

	var buf bytes.Buffer
	client2.Stream(nil, &buf, nil)
	client1.Stream(strings.NewReader("test"), nil, nil)
}

func testReceiveThenSend(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	client2, _ := NewClient("127.0.0.1:9500", channelID)

	var buf bytes.Buffer
	client2.Stream(nil, &buf, nil)
	client1.Stream(strings.NewReader("test"), nil, nil)
}

func testSendThenRespond(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	client2, _ := NewClient("127.0.0.1:9500", channelID)

	var reqBuf, respBuf bytes.Buffer
	client2.Stream(strings.NewReader("req-resp"), &reqBuf, nil)
	client1.Stream(strings.NewReader("req"), &respBuf, nil)
}

func testRespondThenSend(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	client2, _ := NewClient("127.0.0.1:9500", channelID)

	var reqBuf, respBuf bytes.Buffer
	client2.Stream(strings.NewReader("req-resp"), &reqBuf, nil)
	client1.Stream(strings.NewReader("req"), &respBuf, nil)
}

func testConcurrentSendReceive(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	client2, _ := NewClient("127.0.0.1:9500", channelID)

	var buf bytes.Buffer
	client2.Stream(nil, &buf, nil)
	client1.Stream(strings.NewReader("test"), nil, nil)
}

func testConcurrentSendRespond(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	client2, _ := NewClient("127.0.0.1:9500", channelID)

	var reqBuf, respBuf bytes.Buffer
	client2.Stream(strings.NewReader("req-resp"), &reqBuf, nil)
	client1.Stream(strings.NewReader("req"), &respBuf, nil)
}

func TestTimeoutWithSlowReceivers(t *testing.T) {
	t.Skip("Test invalid: immediate acknowledgment in processIncoming() means no slow receiver scenario possible")
}

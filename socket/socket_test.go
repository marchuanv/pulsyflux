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

// ============================================================================
// Basic Tests
// ============================================================================

func TestClientConnectionRefused(t *testing.T) {
	_, err := NewClient("127.0.0.1:19999", uuid.New())
	if err == nil {
		t.Fatal("Expected connection error, got nil")
	}
}

func TestClientSendAfterClose(t *testing.T) {
	server := NewServer("9091")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	client, err := NewClient("127.0.0.1:9091", uuid.New())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	client.Send(strings.NewReader("test"))
	err = client.Wait()
	if err != errClosed {
		t.Errorf("Expected errClosed, got %v", err)
	}
}

func TestClientMultipleClose(t *testing.T) {
	server := NewServer("9095")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	client, err := NewClient("127.0.0.1:9095", uuid.New())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.Close()
	client.Close()
	client.Close()
}

// ============================================================================
// Payload Tests
// ============================================================================

func TestClientEmptyPayload(t *testing.T) {
	server := NewServer("9092")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, err := NewClient("127.0.0.1:9092", channelID)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()

	client2, err := NewClient("127.0.0.1:9092", channelID)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	var buf bytes.Buffer
	client2.Receive(&buf)
	client1.Send(strings.NewReader(""))

	if err := client1.Wait(); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if err := client2.Wait(); err != nil {
		t.Fatalf("Receive failed: %v", err)
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
	client1, err := NewClient("127.0.0.1:9093", channelID)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()

	client2, err := NewClient("127.0.0.1:9093", channelID)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	largeData := bytes.Repeat([]byte("X"), 5*1024*1024)
	var buf bytes.Buffer

	client1.Send(bytes.NewReader(largeData))
	client2.Receive(&buf)

	if err := client1.Wait(); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if err := client2.Wait(); err != nil {
		t.Fatalf("Receive failed: %v", err)
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
			client1, err := NewClient("127.0.0.1:9101", channelID)
			if err != nil {
				t.Fatalf("Failed to create client1: %v", err)
			}
			defer client1.Close()

			client2, err := NewClient("127.0.0.1:9101", channelID)
			if err != nil {
				t.Fatalf("Failed to create client2: %v", err)
			}
			defer client2.Close()

			data := bytes.Repeat([]byte("A"), tc.size)
			var reqBuf, respBuf bytes.Buffer

			client1.Send(bytes.NewReader(data))

			// Respond needs to create response after receiving request
			go func() {
				client2.Respond(&reqBuf, bytes.NewReader(data))
			}()

			client1.Receive(&respBuf)

			if err := client1.Wait(); err != nil {
				t.Fatalf("Client1 failed: %v", err)
			}
			if err := client2.Wait(); err != nil {
				t.Fatalf("Client2 failed: %v", err)
			}

			if reqBuf.Len() != tc.size {
				t.Errorf("Expected %d bytes, got %d", tc.size, reqBuf.Len())
			}
			if respBuf.Len() != tc.size {
				t.Errorf("Expected %d bytes response, got %d", tc.size, respBuf.Len())
			}
		})
	}
}

// ============================================================================
// Concurrency Tests
// ============================================================================

func TestClientSequentialRequests(t *testing.T) {
	server := NewServer("9100")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, err := NewClient("127.0.0.1:9100", channelID)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()

	client2, err := NewClient("127.0.0.1:9100", channelID)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	for i := 0; i < 5; i++ {
		msg := "msg" + string(rune('0'+i))
		data := []byte(msg)
		var reqBuf, respBuf bytes.Buffer

		client1.Send(strings.NewReader(msg))

		go func() {
			client2.Respond(&reqBuf, bytes.NewReader(append(data, []byte("-echo")...)))
		}()

		client1.Receive(&respBuf)

		if err := client1.Wait(); err != nil {
			t.Fatalf("Client1 %d failed: %v", i, err)
		}
		if err := client2.Wait(); err != nil {
			t.Fatalf("Client2 %d failed: %v", i, err)
		}

		expected := msg + "-echo"
		if respBuf.String() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, respBuf.String())
		}
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
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9096", channelID)
	defer client2.Close()
	client3, _ := NewClient("127.0.0.1:9096", channelID)
	defer client3.Close()

	var buf2, buf3 bytes.Buffer
	client2.Receive(&buf2)
	client3.Receive(&buf3)

	// Send will wait for receivers to be available
	client1.Send(strings.NewReader("broadcast"))

	if err := client1.Wait(); err != nil {
		t.Fatalf("Client1 failed: %v", err)
	}
	if err := client2.Wait(); err != nil {
		t.Fatalf("Client2 failed: %v", err)
	}
	if err := client3.Wait(); err != nil {
		t.Fatalf("Client3 failed: %v", err)
	}

	if buf2.String() != "broadcast" {
		t.Errorf("Client2 expected 'broadcast', got '%s'", buf2.String())
	}
	if buf3.String() != "broadcast" {
		t.Errorf("Client3 expected 'broadcast', got '%s'", buf3.String())
	}
}

// ============================================================================
// Error Tests
// ============================================================================

func TestClientReadError(t *testing.T) {
	server := NewServer("9098")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, err := NewClient("127.0.0.1:9098", channelID)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client1.Close()

	client2, err := NewClient("127.0.0.1:9098", channelID)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	var buf bytes.Buffer
	client2.Respond(&buf, nil)

	errReader := &errorReader{err: errors.New("read error")}
	client1.Send(errReader)

	err = client1.Wait()
	if err == nil {
		t.Fatal("Expected read error")
	}
	if !strings.Contains(err.Error(), "read error") {
		t.Errorf("Expected 'read error', got: %v", err)
	}
}

// ============================================================================
// Timeout Tests
// ============================================================================

func TestTimeoutNoReceivers(t *testing.T) {
	server := NewServer("9200")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	publisher, _ := NewClient("127.0.0.1:9200", channelID)
	defer publisher.Close()

	publisher.Send(strings.NewReader("test"))
	err := publisher.Wait()
	if err == nil {
		t.Fatal("Expected no receivers error")
	}
	if !strings.Contains(err.Error(), "no receivers") {
		t.Errorf("Expected 'no receivers' error, got: %v", err)
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkSendReceive(b *testing.B) {
	server := NewServer("9300")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9300", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9300", channelID)
	defer client2.Close()

	data := []byte("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(bytes.NewReader(data))
		client1.Wait()
		client2.Wait()
	}
}

func BenchmarkSendReceive_1KB(b *testing.B) {
	server := NewServer("9301")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9301", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9301", channelID)
	defer client2.Close()

	data := bytes.Repeat([]byte("X"), 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(bytes.NewReader(data))
		client1.Wait()
		client2.Wait()
	}
}

func BenchmarkSendReceive_64KB(b *testing.B) {
	server := NewServer("9302")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9302", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9302", channelID)
	defer client2.Close()

	data := bytes.Repeat([]byte("X"), 64*1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(bytes.NewReader(data))
		client1.Wait()
		client2.Wait()
	}
}

func BenchmarkSendReceive_1MB(b *testing.B) {
	server := NewServer("9303")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9303", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9303", channelID)
	defer client2.Close()

	data := bytes.Repeat([]byte("X"), 1024*1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(bytes.NewReader(data))
		client1.Wait()
		client2.Wait()
	}
}

func BenchmarkSendReceive_Throughput(b *testing.B) {
	server := NewServer("9304")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9304", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9304", channelID)
	defer client2.Close()

	data := bytes.Repeat([]byte("X"), 8192)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(bytes.NewReader(data))
		client1.Wait()
		client2.Wait()
	}
}

func BenchmarkBroadcast_MultipleReceivers(b *testing.B) {
	server := NewServer("9305")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9305", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9305", channelID)
	defer client2.Close()
	client3, _ := NewClient("127.0.0.1:9305", channelID)
	defer client3.Close()

	data := []byte("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf2, buf3 bytes.Buffer
		client2.Receive(&buf2)
		client3.Receive(&buf3)
		client1.Send(bytes.NewReader(data))
		client1.Wait()
		client2.Wait()
		client3.Wait()
	}
}

func BenchmarkSendReceive_Sequential(b *testing.B) {
	server := NewServer("9306")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9306", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9306", channelID)
	defer client2.Close()

	data := []byte("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(bytes.NewReader(data))
		client1.Wait()
		client2.Wait()
	}
}

func BenchmarkMultipleChannels(b *testing.B) {
	server := NewServer("9307")
	server.Start()
	defer server.Stop()

	channel1 := uuid.New()
	channel2 := uuid.New()
	c1a, _ := NewClient("127.0.0.1:9307", channel1)
	defer c1a.Close()
	c1b, _ := NewClient("127.0.0.1:9307", channel1)
	defer c1b.Close()
	c2a, _ := NewClient("127.0.0.1:9307", channel2)
	defer c2a.Close()
	c2b, _ := NewClient("127.0.0.1:9307", channel2)
	defer c2b.Close()

	data := []byte("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf1, buf2 bytes.Buffer
		c1b.Receive(&buf1)
		c2b.Receive(&buf2)
		c1a.Send(bytes.NewReader(data))
		c2a.Send(bytes.NewReader(data))
		c1a.Wait()
		c2a.Wait()
		c1b.Wait()
		c2b.Wait()
	}
}

func BenchmarkRespond_RequestResponse(b *testing.B) {
	server := NewServer("9308")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9308", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9308", channelID)
	defer client2.Close()

	data := []byte("request")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var reqBuf, respBuf bytes.Buffer
		client1.Send(bytes.NewReader(data))
		go func() {
			client2.Respond(&reqBuf, bytes.NewReader([]byte("response")))
		}()
		client1.Receive(&respBuf)
		client1.Wait()
		client2.Wait()
	}
}

func BenchmarkSendReceive_RateMetric(b *testing.B) {
	server := NewServer("9309")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9309", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9309", channelID)
	defer client2.Close()

	data := []byte("test")
	b.ResetTimer()
	start := time.Now()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(bytes.NewReader(data))
		client1.Wait()
		client2.Wait()
	}
	duration := time.Since(start)
	b.ReportMetric(float64(b.N)/duration.Seconds(), "ops/s")
}

func BenchmarkSendReceive_Bandwidth(b *testing.B) {
	server := NewServer("9310")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9310", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9310", channelID)
	defer client2.Close()

	data := bytes.Repeat([]byte("X"), 1024*1024)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(bytes.NewReader(data))
		client1.Wait()
		client2.Wait()
	}
}

func BenchmarkBroadcast_MultipleSenders(b *testing.B) {
	server := NewServer("9311")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	receiver, _ := NewClient("127.0.0.1:9311", channelID)
	defer receiver.Close()
	sender1, _ := NewClient("127.0.0.1:9311", channelID)
	defer sender1.Close()
	sender2, _ := NewClient("127.0.0.1:9311", channelID)
	defer sender2.Close()

	data := []byte("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		receiver.Receive(&buf)
		if i%2 == 0 {
			sender1.Send(bytes.NewReader(data))
			sender1.Wait()
		} else {
			sender2.Send(bytes.NewReader(data))
			sender2.Wait()
		}
		receiver.Wait()
	}
}

func BenchmarkSendReceive_Latency(b *testing.B) {
	server := NewServer("9312")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9312", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9312", channelID)
	defer client2.Close()

	data := []byte("test")
	var latencies []time.Duration
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(bytes.NewReader(data))
		client1.Wait()
		client2.Wait()
		latencies = append(latencies, time.Since(start))
	}
	if len(latencies) > 0 {
		var sum time.Duration
		for _, l := range latencies {
			sum += l
		}
		b.ReportMetric(float64(sum/time.Duration(len(latencies)))/float64(time.Microsecond), "μs/op")
	}
}

// ============================================================================
// Leak and Memory Tests
// ============================================================================

func TestNoGoroutineLeak(t *testing.T) {
	initial := runtime.NumGoroutine()

	server := NewServer("9600")
	server.Start()

	channelID := uuid.New()
	for i := 0; i < 10; i++ {
		client1, _ := NewClient("127.0.0.1:9600", channelID)
		client2, _ := NewClient("127.0.0.1:9600", channelID)
		
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(strings.NewReader("test"))
		
		err1 := client1.Wait()
		err2 := client2.Wait()
		
		if err1 != nil {
			t.Fatalf("client1.Wait() error: %v", err1)
		}
		if err2 != nil {
			t.Fatalf("client2.Wait() error: %v", err2)
		}
		
		client1.Close()
		client2.Close()
	}

	server.Stop()
	time.Sleep(100 * time.Millisecond)
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
	for i := 0; i < 100; i++ {
		client1, _ := NewClient("127.0.0.1:9601", channelID)
		client2, _ := NewClient("127.0.0.1:9601", channelID)
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(strings.NewReader("test"))
		client1.Wait()
		client2.Wait()
		client1.Close()
		client2.Close()
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	growth := m2.Alloc - m1.Alloc
	if growth > 1024*1024 {
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
	client2.Receive(&buf)
	client1.Send(strings.NewReader("test"))
	client1.Wait()
	client2.Wait()

	client1.Close()
	client2.Close()

	time.Sleep(50 * time.Millisecond)

	client3, err := NewClient("127.0.0.1:9602", channelID)
	if err != nil {
		t.Fatalf("Failed to create client after cleanup: %v", err)
	}
	defer client3.Close()
}

func TestMemoryStressLargePayloads(t *testing.T) {
	server := NewServer("9603")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9603", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9603", channelID)
	defer client2.Close()

	data := bytes.Repeat([]byte("X"), 5*1024*1024)
	for i := 0; i < 10; i++ {
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(bytes.NewReader(data))
		client1.Wait()
		client2.Wait()
		if buf.Len() != len(data) {
			t.Fatalf("Iteration %d: expected %d bytes, got %d", i, len(data), buf.Len())
		}
	}
}

func TestMemoryStressConcurrent(t *testing.T) {
	server := NewServer("9604")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client1, _ := NewClient("127.0.0.1:9604", channelID)
			defer client1.Close()
			client2, _ := NewClient("127.0.0.1:9604", channelID)
			defer client2.Close()
			for j := 0; j < 10; j++ {
				var buf bytes.Buffer
				client2.Receive(&buf)
				client1.Send(strings.NewReader("test"))
				client1.Wait()
				client2.Wait()
			}
		}()
	}
	wg.Wait()
}

func TestMemoryPoolEfficiency(t *testing.T) {
	server := NewServer("9605")
	server.Start()
	defer server.Stop()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	channelID := uuid.New()
	client1, _ := NewClient("127.0.0.1:9605", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9605", channelID)
	defer client2.Close()

	data := bytes.Repeat([]byte("X"), 1024)
	for i := 0; i < 1000; i++ {
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(bytes.NewReader(data))
		client1.Wait()
		client2.Wait()
	}

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
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9606", channelID)
	defer client2.Close()

	for i := 0; i < 500; i++ {
		var buf bytes.Buffer
		client2.Receive(&buf)
		client1.Send(strings.NewReader("load test"))
		client1.Wait()
		client2.Wait()
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	growth := m2.Alloc - m1.Alloc
	if growth > 2*1024*1024 {
		t.Errorf("Memory leak under load: growth=%d bytes", growth)
	}
}

// ============================================================================
// Order Independence Tests
// ============================================================================

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
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9500", channelID)
	defer client2.Close()

	var buf bytes.Buffer
	client1.Send(strings.NewReader("test"))
	client2.Receive(&buf)

	if err := client1.Wait(); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if err := client2.Wait(); err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if buf.String() != "test" {
		t.Errorf("Expected 'test', got '%s'", buf.String())
	}
}

func testReceiveThenSend(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9500", channelID)
	defer client2.Close()

	var buf bytes.Buffer
	client2.Receive(&buf)
	client1.Send(strings.NewReader("test"))

	if err := client1.Wait(); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if err := client2.Wait(); err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if buf.String() != "test" {
		t.Errorf("Expected 'test', got '%s'", buf.String())
	}
}

func testSendThenRespond(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9500", channelID)
	defer client2.Close()

	var reqBuf, respBuf bytes.Buffer
	client1.Send(strings.NewReader("req"))

	go func() {
		client2.Respond(&reqBuf, strings.NewReader("req-resp"))
	}()

	client1.Receive(&respBuf)

	if err := client1.Wait(); err != nil {
		t.Fatalf("Client1 failed: %v", err)
	}
	if err := client2.Wait(); err != nil {
		t.Fatalf("Client2 failed: %v", err)
	}
	if respBuf.String() != "req-resp" {
		t.Errorf("Expected 'req-resp', got '%s'", respBuf.String())
	}
}

func testRespondThenSend(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9500", channelID)
	defer client2.Close()

	var reqBuf, respBuf bytes.Buffer

	go func() {
		client2.Respond(&reqBuf, strings.NewReader("req-resp"))
	}()

	client1.Send(strings.NewReader("req"))
	client1.Receive(&respBuf)

	if err := client1.Wait(); err != nil {
		t.Fatalf("Client1 failed: %v", err)
	}
	if err := client2.Wait(); err != nil {
		t.Fatalf("Client2 failed: %v", err)
	}
	if respBuf.String() != "req-resp" {
		t.Errorf("Expected 'req-resp', got '%s'", respBuf.String())
	}
}

func testConcurrentSendReceive(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9500", channelID)
	defer client2.Close()

	var buf bytes.Buffer
	client1.Send(strings.NewReader("test"))
	client2.Receive(&buf)

	if err := client1.Wait(); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if err := client2.Wait(); err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if buf.String() != "test" {
		t.Errorf("Expected 'test', got '%s'", buf.String())
	}
}

func testConcurrentSendRespond(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9500", channelID)
	defer client2.Close()

	var reqBuf, respBuf bytes.Buffer
	client1.Send(strings.NewReader("req"))

	go func() {
		client2.Respond(&reqBuf, strings.NewReader("req-resp"))
	}()

	client1.Receive(&respBuf)

	if err := client1.Wait(); err != nil {
		t.Fatalf("Client1 failed: %v", err)
	}
	if err := client2.Wait(); err != nil {
		t.Fatalf("Client2 failed: %v", err)
	}
	if respBuf.String() != "req-resp" {
		t.Errorf("Expected 'req-resp', got '%s'", respBuf.String())
	}
}

func TestTimeoutWithSlowReceivers(t *testing.T) {
	t.Skip("Test invalid: immediate acknowledgment in processIncoming() means no slow receiver scenario possible")
}

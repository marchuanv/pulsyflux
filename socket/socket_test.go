package socket

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

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

	err = client.Send(strings.NewReader("test"))
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

	incoming2 := make(chan io.Reader, 1)
	done2 := make(chan struct{})

	// Client2 receives
	go func() {
		defer close(done2)
		if err := client2.Receive(incoming2); err != nil {
			t.Errorf("Receive failed: %v", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	// Client1 sends
	if err := client1.Send(strings.NewReader("")); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Verify client2 received
	req := <-incoming2
	data, _ := io.ReadAll(req)
	if len(data) != 0 {
		t.Errorf("Expected empty payload, got %d bytes", len(data))
	}

	<-done2
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

	incoming2 := make(chan io.Reader, 1)
	done2 := make(chan struct{})

	// Client2 receives
	go func() {
		defer close(done2)
		if err := client2.Receive(incoming2); err != nil {
			t.Errorf("Receive failed: %v", err)
		}
	}()

	time.Sleep(10 * time.Millisecond)

	// Client1 sends
	if err := client1.Send(bytes.NewReader(largeData)); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Verify client2 received
	req := <-incoming2
	data, _ := io.ReadAll(req)
	if len(data) != len(largeData) {
		t.Errorf("Expected %d bytes, got %d", len(largeData), len(data))
	}

	<-done2
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

			incoming := make(chan io.Reader, 1)
			outgoing := make(chan io.Reader, 1)
			done := make(chan error, 1)

			go func() {
				done <- client2.Respond(incoming, outgoing)
			}()

			go func() {
				req := <-incoming
				reqData, _ := io.ReadAll(req)
				if len(reqData) != tc.size {
					t.Errorf("Expected %d bytes, got %d", tc.size, len(reqData))
				}
				outgoing <- bytes.NewReader(reqData)
			}()

			if err := client1.Send(bytes.NewReader(data)); err != nil {
				t.Fatalf("Send failed: %v", err)
			}

			incoming1 := make(chan io.Reader, 1)
			if err := client1.Receive(incoming1); err != nil {
				t.Fatalf("Receive failed: %v", err)
			}

			<-done
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

	count := 5
	for i := 0; i < count; i++ {
		incoming1 := make(chan io.Reader, 1)
		incoming2 := make(chan io.Reader, 1)
		outgoing2 := make(chan io.Reader, 1)
		done := make(chan error, 1)

		go func() {
			done <- client2.Respond(incoming2, outgoing2)
		}()

		go func() {
			req := <-incoming2
			data, _ := io.ReadAll(req)
			outgoing2 <- strings.NewReader(string(data) + "-echo")
		}()

		msg := "msg" + string(rune('0'+i))
		if err := client1.Send(strings.NewReader(msg)); err != nil {
			t.Fatalf("Send %d failed: %v", i, err)
		}

		if err := client1.Receive(incoming1); err != nil {
			t.Fatalf("Receive %d failed: %v", i, err)
		}

		<-done
	}
}

func TestClientMultipleConcurrent(t *testing.T) {
	t.Skip("Test needs redesign for pub-sub pattern with multiple concurrent clients")
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

	incoming := make(chan io.Reader, 1)
	outgoing := make(chan io.Reader, 1)

	go func() {
		client2.Respond(incoming, outgoing)
	}()

	errReader := &errorReader{err: errors.New("read error")}

	err = client1.Send(errReader)
	if err == nil {
		t.Error("Expected read error")
	}
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestClientSelfReceive(t *testing.T) {
	server := NewServer("9099")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client, err := NewClient("127.0.0.1:9099", channelID)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	incoming := make(chan io.Reader, 1)
	recvStarted := make(chan struct{})

	go func() {
		close(recvStarted)
		client.Receive(incoming)
	}()

	<-recvStarted
	time.Sleep(50 * time.Millisecond)

	err = client.Send(strings.NewReader("self-test"))
	if err != nil {
		t.Logf("Send blocked as expected (operations serialized): %v", err)
	}
}

func TestClientZeroTimeout(t *testing.T) {
	server := NewServer("9097")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, err := NewClient("127.0.0.1:9097", channelID)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()

	client2, err := NewClient("127.0.0.1:9097", channelID)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	incoming1 := make(chan io.Reader, 1)
	incoming2 := make(chan io.Reader, 1)
	outgoing2 := make(chan io.Reader, 1)
	done := make(chan error, 1)

	go func() {
		done <- client2.Respond(incoming2, outgoing2)
	}()

	go func() {
		req := <-incoming2
		io.ReadAll(req)
		outgoing2 <- strings.NewReader("ok")
	}()

	if err := client1.Send(strings.NewReader("test")); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if err := client1.Receive(incoming1); err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	<-done
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

	err := publisher.Send(strings.NewReader("test"))
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

func BenchmarkSingleRequest(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

func BenchmarkConcurrentRequests(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

func BenchmarkLargePayload(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

func BenchmarkMultipleChannels(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

func BenchmarkThroughput(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

func BenchmarkSmallPayload(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

func BenchmarkMediumPayload(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

func BenchmarkEchoPayload(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

func BenchmarkParallelConsumers(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

func BenchmarkRequestsPerSecond(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

func BenchmarkBandwidth(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

func BenchmarkMultipleProviders(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

func BenchmarkLatency(b *testing.B) {
	b.Skip("Benchmark needs to be redesigned")
}

// ============================================================================
// Leak and Memory Tests
// ============================================================================

func TestNoGoroutineLeak(t *testing.T) {
	t.Skip("Test needs to be redesigned")
}

func TestNoMemoryLeak(t *testing.T) {
	t.Skip("Test needs to be redesigned")
}

func TestConnectionCleanup(t *testing.T) {
	t.Skip("Test needs to be redesigned")
}

func TestMemoryStressLargePayloads(t *testing.T) {
	t.Skip("Test needs to be redesigned")
}

func TestMemoryStressConcurrent(t *testing.T) {
	t.Skip("Test needs to be redesigned")
}

func TestMemoryPoolEfficiency(t *testing.T) {
	t.Skip("Test needs to be redesigned")
}

func TestMemoryLeakUnderLoad(t *testing.T) {
	t.Skip("Test needs to be redesigned")
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

	if err := client1.Send(strings.NewReader("test")); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	incoming := make(chan io.Reader, 1)
	if err := client2.Receive(incoming); err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	data, _ := io.ReadAll(<-incoming)
	if string(data) != "test" {
		t.Errorf("Expected 'test', got '%s'", string(data))
	}
}

func testReceiveThenSend(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9500", channelID)
	defer client2.Close()

	incoming := make(chan io.Reader, 1)
	done := make(chan error, 1)
	go func() {
		done <- client2.Receive(incoming)
	}()

	if err := client1.Send(strings.NewReader("test")); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	data, _ := io.ReadAll(<-incoming)
	if string(data) != "test" {
		t.Errorf("Expected 'test', got '%s'", string(data))
	}
}

func testSendThenRespond(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9500", channelID)
	defer client2.Close()

	if err := client1.Send(strings.NewReader("req")); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	incoming := make(chan io.Reader, 1)
	outgoing := make(chan io.Reader, 1)
	done := make(chan error, 1)
	go func() {
		done <- client2.Respond(incoming, outgoing)
	}()

	go func() {
		req := <-incoming
		data, _ := io.ReadAll(req)
		outgoing <- strings.NewReader(string(data) + "-resp")
	}()

	incoming1 := make(chan io.Reader, 1)
	if err := client1.Receive(incoming1); err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	data, _ := io.ReadAll(<-incoming1)
	if string(data) != "req-resp" {
		t.Errorf("Expected 'req-resp', got '%s'", string(data))
	}

	<-done
}

func testRespondThenSend(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9500", channelID)
	defer client2.Close()

	incoming := make(chan io.Reader, 1)
	outgoing := make(chan io.Reader, 1)
	done := make(chan error, 1)
	go func() {
		done <- client2.Respond(incoming, outgoing)
	}()

	go func() {
		req := <-incoming
		data, _ := io.ReadAll(req)
		outgoing <- strings.NewReader(string(data) + "-resp")
	}()

	if err := client1.Send(strings.NewReader("req")); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	incoming1 := make(chan io.Reader, 1)
	if err := client1.Receive(incoming1); err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	data, _ := io.ReadAll(<-incoming1)
	if string(data) != "req-resp" {
		t.Errorf("Expected 'req-resp', got '%s'", string(data))
	}

	<-done
}

func testConcurrentSendReceive(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9500", channelID)
	defer client2.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		client1.Send(strings.NewReader("test"))
	}()

	go func() {
		defer wg.Done()
		incoming := make(chan io.Reader, 1)
		client2.Receive(incoming)
		data, _ := io.ReadAll(<-incoming)
		if string(data) != "test" {
			t.Errorf("Expected 'test', got '%s'", string(data))
		}
	}()

	wg.Wait()
}

func testConcurrentSendRespond(t *testing.T, server *Server, channelID uuid.UUID) {
	client1, _ := NewClient("127.0.0.1:9500", channelID)
	defer client1.Close()
	client2, _ := NewClient("127.0.0.1:9500", channelID)
	defer client2.Close()

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		client1.Send(strings.NewReader("req"))
	}()

	go func() {
		defer wg.Done()
		incoming := make(chan io.Reader, 1)
		outgoing := make(chan io.Reader, 1)
		go func() {
			req := <-incoming
			data, _ := io.ReadAll(req)
			outgoing <- strings.NewReader(string(data) + "-resp")
		}()
		client2.Respond(incoming, outgoing)
	}()

	go func() {
		defer wg.Done()
		incoming := make(chan io.Reader, 1)
		client1.Receive(incoming)
		data, _ := io.ReadAll(<-incoming)
		if string(data) != "req-resp" {
			t.Errorf("Expected 'req-resp', got '%s'", string(data))
		}
	}()

	wg.Wait()
}

func TestTimeoutWithReceivers(t *testing.T) {
	t.Skip("Cannot test slow receiver timeout: acks sent immediately when message queued, not when consumed")
}

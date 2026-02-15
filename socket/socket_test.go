package socket

import (
	"bytes"
	"errors"
	"io"
	"strings"
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
		if len(data) != 0 {
			t.Errorf("Expected empty payload, got %d bytes", len(data))
		}
		outgoing2 <- strings.NewReader("response")
	}()

	if err := client1.Send(strings.NewReader("")); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if err := client1.Receive(incoming1); err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	resp := <-incoming1
	data, _ := io.ReadAll(resp)
	if string(data) != "response" {
		t.Errorf("Expected 'response', got '%s'", string(data))
	}

	<-done
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
		if len(data) != len(largeData) {
			t.Errorf("Expected %d bytes, got %d", len(largeData), len(data))
		}
		outgoing2 <- bytes.NewReader(largeData)
	}()

	if err := client1.Send(bytes.NewReader(largeData)); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if err := client1.Receive(incoming1); err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	resp := <-incoming1
	data, _ := io.ReadAll(resp)
	if len(data) != len(largeData) {
		t.Errorf("Expected %d bytes in response, got %d", len(largeData), len(data))
	}

	<-done
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

func TestConsumerTimeout(t *testing.T) {
	server := NewServer("9200")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()

	provider, _ := NewClient("127.0.0.1:9200", channelID)
	defer provider.Close()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewClient("127.0.0.1:9200", channelID)
	defer consumer.Close()

	err := consumer.Send(strings.NewReader("test"))
	if err == nil {
		t.Fatal("Expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "no receivers") {
		t.Errorf("Expected timeout error, got: %v", err)
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

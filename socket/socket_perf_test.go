package socket

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func BenchmarkSmallPayload(b *testing.B) {
	server := NewServer("9300")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewClient("127.0.0.1:9300", channelID)
	defer provider.Close()

	go func() {
		for {
			_, _, ok := provider.Receive()
			if !ok {
				break
			}
			provider.Send(strings.NewReader("ok"), 5*time.Second)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewClient("127.0.0.1:9300", channelID)
	defer consumer.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		consumer.Send(strings.NewReader("test"), 5*time.Second)
	}
}

func BenchmarkMediumPayload(b *testing.B) {
	server := NewServer("9301")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewClient("127.0.0.1:9301", channelID)
	defer provider.Close()

	mediumData := strings.Repeat("X", 10*1024) // 10KB

	go func() {
		for {
			_, _, ok := provider.Receive()
			if !ok {
				break
			}
			provider.Send(strings.NewReader(mediumData), 5*time.Second)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewClient("127.0.0.1:9301", channelID)
	defer consumer.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		consumer.Send(strings.NewReader(mediumData), 5*time.Second)
	}
}

func BenchmarkEchoPayload(b *testing.B) {
	server := NewServer("9302")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewClient("127.0.0.1:9302", channelID)
	defer provider.Close()

	go func() {
		for {
			_, r, ok := provider.Receive()
			if !ok {
				break
			}
			data, _ := io.ReadAll(r)
			provider.Send(bytes.NewReader(data), 5*time.Second)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewClient("127.0.0.1:9302", channelID)
	defer consumer.Close()

	payload := strings.Repeat("X", 1024) // 1KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		consumer.Send(strings.NewReader(payload), 5*time.Second)
	}
}

func BenchmarkParallelConsumers(b *testing.B) {
	server := NewServer("9303")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewClient("127.0.0.1:9303", channelID)
	defer provider.Close()

	go func() {
		for {
			_, _, ok := provider.Receive()
			if !ok {
				break
			}
			provider.Send(strings.NewReader("ok"), 5*time.Second)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	numConsumers := 50
	consumers := make([]*Client, numConsumers)
	for i := 0; i < numConsumers; i++ {
		consumers[i], _ = NewClient("127.0.0.1:9303", channelID)
		defer consumers[i].Close()
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		idx := 0
		for pb.Next() {
			consumer := consumers[idx%numConsumers]
			consumer.Send(strings.NewReader("test"), 5*time.Second)
			idx++
		}
	})
}

func BenchmarkRequestsPerSecond(b *testing.B) {
	server := NewServer("9304")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewClient("127.0.0.1:9304", channelID)
	defer provider.Close()

	var processed atomic.Int64

	go func() {
		for {
			_, _, ok := provider.Receive()
			if !ok {
				break
			}
			provider.Send(strings.NewReader("ok"), 5*time.Second)
			processed.Add(1)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewClient("127.0.0.1:9304", channelID)
	defer consumer.Close()

	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		consumer.Send(strings.NewReader("test"), 5*time.Second)
	}

	elapsed := time.Since(start)
	rps := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(rps, "req/s")
}

func BenchmarkBandwidth(b *testing.B) {
	server := NewServer("9305")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewClient("127.0.0.1:9305", channelID)
	defer provider.Close()

	largeData := strings.Repeat("X", 1024*1024) // 1MB

	go func() {
		for {
			_, _, ok := provider.Receive()
			if !ok {
				break
			}
			provider.Send(strings.NewReader(largeData), 5*time.Second)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewClient("127.0.0.1:9305", channelID)
	defer consumer.Close()

	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		consumer.Send(strings.NewReader(largeData), 5*time.Second)
	}

	elapsed := time.Since(start)
	totalBytes := float64(b.N * len(largeData) * 2) // Request + Response
	mbps := (totalBytes / elapsed.Seconds()) / (1024 * 1024)
	b.ReportMetric(mbps, "MB/s")
}

func BenchmarkMultipleProviders(b *testing.B) {
	server := NewServer("9306")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	numProviders := 10
	channelID := uuid.New()
	providers := make([]*Client, numProviders)

	for i := 0; i < numProviders; i++ {
		providers[i], _ = NewClient("127.0.0.1:9306", channelID)
		defer providers[i].Close()

		go func(p *Client) {
			for {
				_, _, ok := p.Receive()
				if !ok {
					break
				}
				p.Send(strings.NewReader("ok"), 5*time.Second)
			}
		}(providers[i])
	}

	time.Sleep(50 * time.Millisecond)

	numConsumers := 20
	consumers := make([]*Client, numConsumers)
	for i := 0; i < numConsumers; i++ {
		consumers[i], _ = NewClient("127.0.0.1:9306", channelID)
		defer consumers[i].Close()
	}

	b.ResetTimer()
	var wg sync.WaitGroup
	for i := 0; i < numConsumers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < b.N/numConsumers; j++ {
				consumers[idx].Send(strings.NewReader("test"), 5*time.Second)
			}
		}(i)
	}
	wg.Wait()
}

func BenchmarkLatency(b *testing.B) {
	server := NewServer("9307")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewClient("127.0.0.1:9307", channelID)
	defer provider.Close()

	go func() {
		for {
			_, _, ok := provider.Receive()
			if !ok {
				break
			}
			provider.Send(strings.NewReader("ok"), 5*time.Second)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	consumer, _ := NewClient("127.0.0.1:9307", channelID)
	defer consumer.Close()

	var totalLatency time.Duration
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		consumer.Send(strings.NewReader("test"), 5*time.Second)
		totalLatency += time.Since(start)
	}

	avgLatency := totalLatency / time.Duration(b.N)
	b.ReportMetric(float64(avgLatency.Microseconds()), "µs/op")
}

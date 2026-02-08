package socket

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func BenchmarkSingleRequest(b *testing.B) {
	server := NewServer("9095")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewClient("127.0.0.1:9095", channelID, roleProvider)
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

	consumer, _ := NewClient("127.0.0.1:9095", channelID, roleConsumer)
	defer consumer.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		consumer.Send(strings.NewReader("test"), 5*time.Second)
	}
}

func BenchmarkConcurrentRequests(b *testing.B) {
	server := NewServer("9096")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewClient("127.0.0.1:9096", channelID, roleProvider)
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

	numConsumers := 10
	consumers := make([]*Client, numConsumers)
	for i := 0; i < numConsumers; i++ {
		consumers[i], _ = NewClient("127.0.0.1:9096", channelID, roleConsumer)
		defer consumers[i].Close()
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		consumerIdx := 0
		for pb.Next() {
			consumer := consumers[consumerIdx%numConsumers]
			consumer.Send(strings.NewReader("test"), 5*time.Second)
			consumerIdx++
		}
	})
}

func BenchmarkLargePayload(b *testing.B) {
	server := NewServer("9097")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewClient("127.0.0.1:9097", channelID, roleProvider)
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

	consumer, _ := NewClient("127.0.0.1:9097", channelID, roleConsumer)
	defer consumer.Close()

	largeData := strings.Repeat("X", 1024*1024) // 1MB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		consumer.Send(strings.NewReader(largeData), 5*time.Second)
	}
}

func BenchmarkMultipleChannels(b *testing.B) {
	server := NewServer("9098")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	numChannels := 10
	channels := make([]uuid.UUID, numChannels)
	providers := make([]*Client, numChannels)
	consumers := make([]*Client, numChannels)

	for i := 0; i < numChannels; i++ {
		channels[i] = uuid.New()
		providers[i], _ = NewClient("127.0.0.1:9098", channels[i], roleProvider)
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

	for i := 0; i < numChannels; i++ {
		consumers[i], _ = NewClient("127.0.0.1:9098", channels[i], roleConsumer)
		defer consumers[i].Close()
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		channelIdx := 0
		for pb.Next() {
			consumer := consumers[channelIdx%numChannels]
			consumer.Send(strings.NewReader("test"), 5*time.Second)
			channelIdx++
		}
	})
}

func BenchmarkThroughput(b *testing.B) {
	server := NewServer("9099")
	server.Start()
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	provider, _ := NewClient("127.0.0.1:9099", channelID, roleProvider)
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

	numConsumers := 100
	consumers := make([]*Client, numConsumers)
	for i := 0; i < numConsumers; i++ {
		consumers[i], _ = NewClient("127.0.0.1:9099", channelID, roleConsumer)
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

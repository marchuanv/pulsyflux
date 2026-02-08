package broker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"pulsyflux/socket"
)

var (
	benchServer     *socket.Server
	benchServerOnce sync.Once
)

func setupBenchServer() {
	benchServerOnce.Do(func() {
		benchServer = socket.NewServer("9200")
		benchServer.Start()
		time.Sleep(100 * time.Millisecond)
	})
}

func BenchmarkSinglePublish(b *testing.B) {
	setupBenchServer()

	brokerChanID := uuid.New()
	broker, _ := NewBroker("127.0.0.1:9200", brokerChanID)
	defer broker.Close()

	ch, _ := broker.Subscribe("bench.topic")
	go func() {
		for range ch {
		}
	}()

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	payload := []byte("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.Publish(ctx, "bench.topic", payload, nil)
	}
	b.StopTimer()
	b.ReportMetric(float64(broker.ConnectionCount()), "conns")
}

func BenchmarkMultipleSubscribers(b *testing.B) {
	setupBenchServer()

	brokerChanID := uuid.New()
	broker, _ := NewBroker("127.0.0.1:9200", brokerChanID)
	defer broker.Close()

	numSubs := 10
	for i := 0; i < numSubs; i++ {
		ch, _ := broker.Subscribe("bench.topic")
		go func() {
			for range ch {
			}
		}()
	}

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	payload := []byte("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.Publish(ctx, "bench.topic", payload, nil)
	}
	b.StopTimer()
	b.ReportMetric(float64(broker.ConnectionCount()), "conns")
	b.ReportMetric(float64(broker.SubscriberCount()), "subs")
}

func BenchmarkLargePayload(b *testing.B) {
	setupBenchServer()

	brokerChanID := uuid.New()
	broker, _ := NewBroker("127.0.0.1:9200", brokerChanID)
	defer broker.Close()

	ch, _ := broker.Subscribe("bench.topic")
	go func() {
		for range ch {
		}
	}()

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	payload := make([]byte, 1024*1024) // 1MB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.Publish(ctx, "bench.topic", payload, nil)
	}
	b.StopTimer()
	b.ReportMetric(float64(broker.ConnectionCount()), "conns")
}

func BenchmarkWithHeaders(b *testing.B) {
	setupBenchServer()

	brokerChanID := uuid.New()
	broker, _ := NewBroker("127.0.0.1:9200", brokerChanID)
	defer broker.Close()

	ch, _ := broker.Subscribe("bench.topic")
	go func() {
		for range ch {
		}
	}()

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	payload := []byte("test")
	headers := map[string]string{
		"user_id":    "123",
		"request_id": "abc",
		"source":     "api",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.Publish(ctx, "bench.topic", payload, headers)
	}
	b.StopTimer()
	b.ReportMetric(float64(broker.ConnectionCount()), "conns")
}

func BenchmarkThroughput(b *testing.B) {
	setupBenchServer()

	brokerChanID := uuid.New()
	broker, _ := NewBroker("127.0.0.1:9200", brokerChanID)
	defer broker.Close()

	numSubs := 5
	for i := 0; i < numSubs; i++ {
		ch, _ := broker.Subscribe("bench.topic")
		go func() {
			for range ch {
			}
		}()
	}

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	payload := []byte("test")

	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		broker.Publish(ctx, "bench.topic", payload, nil)
	}

	b.StopTimer()
	elapsed := time.Since(start)
	msgsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(msgsPerSec, "msg/s")
	b.ReportMetric(float64(broker.ConnectionCount()), "conns")
	b.ReportMetric(float64(broker.SubscriberCount()), "subs")
}

func BenchmarkLatency(b *testing.B) {
	setupBenchServer()

	brokerChanID := uuid.New()
	broker, _ := NewBroker("127.0.0.1:9200", brokerChanID)
	defer broker.Close()

	ch, _ := broker.Subscribe("bench.topic")

	received := make(chan struct{}, 1)
	go func() {
		for range ch {
			select {
			case received <- struct{}{}:
			default:
			}
		}
	}()

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	payload := []byte("test")

	var totalLatency time.Duration
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		broker.Publish(ctx, "bench.topic", payload, nil)
		<-received
		totalLatency += time.Since(start)
	}

	b.StopTimer()
	avgLatency := totalLatency / time.Duration(b.N)
	b.ReportMetric(float64(avgLatency.Microseconds()), "µs/op")
	b.ReportMetric(float64(broker.ConnectionCount()), "conns")
}

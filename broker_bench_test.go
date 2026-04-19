package broker

import (
	"testing"

	"github.com/google/uuid"
)

func BenchmarkPublish(b *testing.B) {
	server := NewServer(":0")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client, _ := NewClient(server.Addr(), channelID)
	payload := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Publish(payload)
	}
}

func BenchmarkPubSub(b *testing.B) {
	server := NewServer(":0")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	client1, _ := NewClient(server.Addr(), channelID)
	client2, _ := NewClient(server.Addr(), channelID)

	ch := client2.Subscribe()
	payload := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client1.Publish(payload)
		<-ch
	}
}

func BenchmarkBroadcast2(b *testing.B) {
	benchmarkBroadcast(b, 2)
}

func BenchmarkBroadcast5(b *testing.B) {
	benchmarkBroadcast(b, 5)
}

func BenchmarkBroadcast10(b *testing.B) {
	benchmarkBroadcast(b, 10)
}

func benchmarkBroadcast(b *testing.B, numClients int) {
	server := NewServer(":0")
	server.Start()
	defer server.Stop()

	channelID := uuid.New()
	clients := make([]*Client, numClients)
	subs := make([]<-chan []byte, numClients)

	for i := 0; i < numClients; i++ {
		clients[i], _ = NewClient(server.Addr(), channelID)
		if i > 0 {
			subs[i] = clients[i].Subscribe()
		}
	}

	payload := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clients[0].Publish(payload)
		for j := 1; j < numClients; j++ {
			<-subs[j]
		}
	}
}

func BenchmarkMultipleChannels(b *testing.B) {
	server := NewServer(":0")
	server.Start()
	defer server.Stop()

	ch1 := uuid.New()
	ch2 := uuid.New()
	ch3 := uuid.New()

	c1, _ := NewClient(server.Addr(), ch1)
	c2, _ := NewClient(server.Addr(), ch2)
	c3, _ := NewClient(server.Addr(), ch3)

	payload := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c1.Publish(payload)
		c2.Publish(payload)
		c3.Publish(payload)
	}
}

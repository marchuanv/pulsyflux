package tcpconn

import (
	"github.com/google/uuid"
	"net"
	"testing"
)

func BenchmarkConnection_SmallMessage(b *testing.B) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	id := uuid.New()
	go func() {
		for {
			conn, _ := listener.Accept()
			if conn == nil {
				return
			}
			wrapped := WrapConnection(conn, id)
			for {
				data, err := wrapped.Receive()
				if err != nil {
					break
				}
				wrapped.Send(data)
			}
		}
	}()

	c := NewConnection(listener.Addr().String(), id)
	data := make([]byte, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Send(data)
		c.Receive()
	}
}

func BenchmarkConnection_LargeMessage(b *testing.B) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	id := uuid.New()
	go func() {
		for {
			conn, _ := listener.Accept()
			if conn == nil {
				return
			}
			wrapped := WrapConnection(conn, id)
			for {
				data, err := wrapped.Receive()
				if err != nil {
					break
				}
				wrapped.Send(data)
			}
		}
	}()

	c := NewConnection(listener.Addr().String(), id)
	data := make([]byte, 1024*1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Send(data)
		c.Receive()
	}
}

func BenchmarkConnection_Chunking(b *testing.B) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	id := uuid.New()
	go func() {
		for {
			conn, _ := listener.Accept()
			if conn == nil {
				return
			}
			wrapped := WrapConnection(conn, id)
			for {
				data, err := wrapped.Receive()
				if err != nil {
					break
				}
				wrapped.Send(data)
			}
		}
	}()

	c := NewConnection(listener.Addr().String(), id)
	data := make([]byte, 200*1024) // 3+ chunks

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Send(data)
		c.Receive()
	}
}

func BenchmarkPool_MultipleConnections(b *testing.B) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	id := uuid.New()
	go func() {
		for {
			conn, _ := listener.Accept()
			if conn == nil {
				return
			}
			wrapped := WrapConnection(conn, id)
			for {
				data, err := wrapped.Receive()
				if err != nil {
					break
				}
				wrapped.Send(data)
			}
		}
	}()

	addr := listener.Addr().String()
	data := make([]byte, 1024)
	c := NewConnection(addr, id)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Send(data)
		c.Receive()
	}
}

func BenchmarkConnection_SendOnly(b *testing.B) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	id := uuid.New()
	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			wrapped := WrapConnection(conn, id)
			for {
				_, err := wrapped.Receive()
				if err != nil {
					break
				}
			}
		}
	}()

	c := NewConnection(listener.Addr().String(), id)
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Send(data)
	}
}

func BenchmarkConnection_ReceiveOnly(b *testing.B) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	id := uuid.New()
	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			wrapped := WrapConnection(conn, id)
			data := make([]byte, 1024)
			for i := 0; i < b.N; i++ {
				wrapped.Send(data)
			}
		}
	}()

	c := NewConnection(listener.Addr().String(), id)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Receive()
	}
}

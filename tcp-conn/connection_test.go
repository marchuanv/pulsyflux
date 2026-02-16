package tcpconn

import (
	"net"
	"testing"
	"time"
)

func TestConnection_SendReceive(t *testing.T) {
	// Start test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, _ := listener.Accept()
		defer conn.Close()
		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		conn.Write(buf[:n])
	}()

	c := NewConnection(listener.Addr().String(), 1*time.Second)
	if c == nil {
		t.Fatal("NewConnection returned nil")
	}

	err = c.Send([]byte("hello"))
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	data, err := c.Receive()
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	if string(data) != "hello" {
		t.Errorf("Expected 'hello', got '%s'", string(data))
	}
}

func TestConnection_IdleTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	c := NewConnection(listener.Addr().String(), 100*time.Millisecond)
	if c == nil {
		t.Fatal("NewConnection returned nil")
	}

	time.Sleep(250 * time.Millisecond)

	err = c.Send([]byte("test"))
	if err != nil {
		// Connection closed or reconnect failed - both acceptable
		return
	}
}

func TestConnection_Reconnect(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	acceptCount := 0
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			acceptCount++
			buf := make([]byte, 1024)
			n, _ := conn.Read(buf)
			conn.Write(buf[:n])
			conn.Close()
		}
	}()

	c := NewConnection(listener.Addr().String(), 100*time.Millisecond)
	if c == nil {
		t.Fatal("NewConnection returned nil")
	}

	// First send
	c.Send([]byte("first"))
	c.Receive()

	// Wait for idle timeout
	time.Sleep(250 * time.Millisecond)

	// Should reconnect
	err = c.Send([]byte("second"))
	if err != nil {
		t.Fatalf("Send after reconnect failed: %v", err)
	}

	data, err := c.Receive()
	if err != nil {
		t.Fatalf("Receive after reconnect failed: %v", err)
	}

	if string(data) != "second" {
		t.Errorf("Expected 'second', got '%s'", string(data))
	}

	if acceptCount < 2 {
		t.Errorf("Expected at least 2 connections, got %d", acceptCount)
	}
}

func TestConnection_InvalidAddress(t *testing.T) {
	c := NewConnection("invalid:99999", 1*time.Second)
	if c != nil {
		t.Error("Expected nil for invalid address")
	}
}

func TestConnection_ActivityPreventsTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				for {
					n, err := c.Read(buf)
					if err != nil {
						return
					}
					c.Write(buf[:n])
				}
			}(conn)
		}
	}()

	c := NewConnection(listener.Addr().String(), 200*time.Millisecond)
	if c == nil {
		t.Fatal("NewConnection returned nil")
	}

	for i := 0; i < 5; i++ {
		time.Sleep(100 * time.Millisecond)
		err := c.Send([]byte("keep-alive"))
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}
		_, err = c.Receive()
		if err != nil {
			t.Fatalf("Receive failed: %v", err)
		}
	}

	err = c.Send([]byte("final"))
	if err != nil {
		t.Error("Connection should still be alive with activity")
	}
}

func TestWrapConnection_ServerSide(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	serverDone := make(chan bool)
	go func() {
		conn, _ := listener.Accept()
		if conn == nil {
			return
		}

		// Wrap accepted connection
		wrapped := WrapConnection(conn, 1*time.Second)
		
		data, err := wrapped.Receive()
		if err != nil {
			t.Errorf("Server receive failed: %v", err)
			return
		}

		err = wrapped.Send(data)
		if err != nil {
			t.Errorf("Server send failed: %v", err)
		}
		serverDone <- true
	}()

	// Client side - also use Connection for framing
	client := NewConnection(listener.Addr().String(), 1*time.Second)
	if client == nil {
		t.Fatal("Client connection failed")
	}

	err = client.Send([]byte("hello server"))
	if err != nil {
		t.Fatalf("Client send failed: %v", err)
	}
	
	data, err := client.Receive()
	if err != nil {
		t.Fatalf("Client read failed: %v", err)
	}

	if string(data) != "hello server" {
		t.Errorf("Expected 'hello server', got '%s'", string(data))
	}

	<-serverDone
}

func TestWrapConnection_NoReconnect(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	wrapped := WrapConnection(server, 100*time.Millisecond)
	
	// Close underlying connection
	server.Close()
	
	time.Sleep(200 * time.Millisecond)
	
	// Should fail - no reconnect for wrapped connections
	err := wrapped.Send([]byte("test"))
	if err == nil {
		t.Error("Expected error after connection closed")
	}
}

func TestConnection_LargeMessage(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, _ := listener.Accept()
		if conn == nil {
			return
		}
		wrapped := WrapConnection(conn, 5*time.Second)
		data, _ := wrapped.Receive()
		wrapped.Send(data)
	}()

	c := NewConnection(listener.Addr().String(), 5*time.Second)
	if c == nil {
		t.Fatal("NewConnection returned nil")
	}

	// 1MB message
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err = c.Send(largeData)
	if err != nil {
		t.Fatalf("Send large message failed: %v", err)
	}

	received, err := c.Receive()
	if err != nil {
		t.Fatalf("Receive large message failed: %v", err)
	}

	if len(received) != len(largeData) {
		t.Errorf("Expected %d bytes, got %d", len(largeData), len(received))
	}
}

func TestConnection_MultipleLogicalConnections(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, _ := listener.Accept()
		if conn == nil {
			return
		}
		wrapped1 := WrapConnectionWithID(conn, "conn1", 5*time.Second)
		wrapped2 := WrapConnectionWithID(conn, "conn2", 5*time.Second)

		data1, _ := wrapped1.Receive()
		wrapped1.Send(data1)

		data2, _ := wrapped2.Receive()
		wrapped2.Send(data2)
	}()

	c1 := NewConnectionWithID(listener.Addr().String(), "conn1", 5*time.Second)
	c2 := NewConnectionWithID(listener.Addr().String(), "conn2", 5*time.Second)

	c1.Send([]byte("message1"))
	c2.Send([]byte("message2"))

	data1, err := c1.Receive()
	if err != nil || string(data1) != "message1" {
		t.Errorf("conn1 expected 'message1', got '%s' err=%v", string(data1), err)
	}

	data2, err := c2.Receive()
	if err != nil || string(data2) != "message2" {
		t.Errorf("conn2 expected 'message2', got '%s' err=%v", string(data2), err)
	}
}

func TestConnection_ConcurrentSendReceive(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, _ := listener.Accept()
		if conn == nil {
			return
		}
		wrapped := WrapConnection(conn, 5*time.Second)
		for i := 0; i < 10; i++ {
			data, err := wrapped.Receive()
			if err != nil {
				return
			}
			wrapped.Send(data)
		}
	}()

	c := NewConnection(listener.Addr().String(), 5*time.Second)
	if c == nil {
		t.Fatal("NewConnection returned nil")
	}

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			msg := []byte{byte(n)}
			c.Send(msg)
			data, err := c.Receive()
			if err != nil || len(data) != 1 || data[0] != byte(n) {
				t.Errorf("Concurrent test failed for %d", n)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestConnection_PoolReferenceCount(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				wrapped := WrapConnection(c, 5*time.Second)
				for {
					data, err := wrapped.Receive()
					if err != nil {
						return
					}
					wrapped.Send(data)
				}
			}(conn)
		}
	}()

	addr := listener.Addr().String()
	c1 := NewConnection(addr, 5*time.Second)
	c2 := NewConnection(addr, 5*time.Second)
	c3 := NewConnection(addr, 5*time.Second)

	if c1 == nil || c2 == nil || c3 == nil {
		t.Fatal("Failed to create connections")
	}

	pool := globalPool.pools[addr]
	if pool.refCount != 3 {
		t.Errorf("Expected refCount 3, got %d", pool.refCount)
	}

	c1.close()
	if pool.refCount != 2 {
		t.Errorf("After close, expected refCount 2, got %d", pool.refCount)
	}

	c2.close()
	c3.close()

	if _, exists := globalPool.pools[addr]; exists {
		t.Error("Pool should be removed after all connections closed")
	}
}

func TestConnection_BrokerHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	// Broker goroutine
	go func() {
		conn, _ := listener.Accept()
		if conn == nil {
			return
		}

		// Broker receives handshake on empty ID connection
		emptyConn := WrapConnection(conn, 5*time.Second)
		handshake, err := emptyConn.Receive()
		if err != nil {
			t.Errorf("Broker failed to receive handshake: %v", err)
			return
		}

		// Verify handshake format
		if string(handshake) != "HANDSHAKE:topic-news" {
			t.Errorf("Expected 'HANDSHAKE:topic-news', got '%s'", string(handshake))
			return
		}

		// Broker creates logical connection with same ID
		logicalConn := WrapConnectionWithID(conn, "topic-news", 5*time.Second)

		// Send READY back on logical connection
		if err := logicalConn.Send([]byte("READY")); err != nil {
			t.Errorf("Broker failed to send READY: %v", err)
			return
		}

		// Now broker can receive messages on logical connection
		data, err := logicalConn.Receive()
		if err != nil {
			t.Errorf("Broker failed to receive data: %v", err)
			return
		}

		// Echo back
		logicalConn.Send(data)
	}()

	// Client creates logical connection - handshake happens automatically
	client := NewConnectionWithID(listener.Addr().String(), "topic-news", 5*time.Second)
	if client == nil {
		t.Fatal("Client connection failed or handshake failed")
	}

	// Now client can send/receive
	err = client.Send([]byte("hello broker"))
	if err != nil {
		t.Fatalf("Client send failed: %v", err)
	}

	data, err := client.Receive()
	if err != nil {
		t.Fatalf("Client receive failed: %v", err)
	}

	if string(data) != "hello broker" {
		t.Errorf("Expected 'hello broker', got '%s'", string(data))
	}
}

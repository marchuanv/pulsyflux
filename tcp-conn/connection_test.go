package tcpconn

import (
	"github.com/google/uuid"
	"net"
	"testing"
)

func TestConnection_SendReceive(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	id := uuid.New().String()
	go func() {
		conn, _ := listener.Accept()
		defer conn.Close()
		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		conn.Write(buf[:n])
	}()

	c := NewConnection(listener.Addr().String(), id)
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

func TestConnection_InvalidAddress(t *testing.T) {
	c := NewConnection("invalid:99999", uuid.New().String())
	if c != nil {
		t.Error("Expected nil for invalid address")
	}
}

func TestWrapConnection_ServerSide(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	clientID := uuid.New().String()
	serverDone := make(chan bool)
	go func() {
		conn, _ := listener.Accept()
		if conn == nil {
			return
		}

		wrapped := WrapConnection(conn, clientID)
		
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

	client := NewConnection(listener.Addr().String(), clientID)
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

func TestConnection_LargeMessage(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	id := uuid.New().String()
	go func() {
		conn, _ := listener.Accept()
		if conn == nil {
			return
		}
		wrapped := WrapConnection(conn, id)
		data, _ := wrapped.Receive()
		wrapped.Send(data)
	}()

	c := NewConnection(listener.Addr().String(), id)
	if c == nil {
		t.Fatal("NewConnection returned nil")
	}

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
		wrapped1 := WrapConnection(conn, "conn1")
		wrapped2 := WrapConnection(conn, "conn2")

		data1, _ := wrapped1.Receive()
		wrapped1.Send(data1)

		data2, _ := wrapped2.Receive()
		wrapped2.Send(data2)
	}()

	c1 := NewConnection(listener.Addr().String(), "conn1")
	c2 := NewConnection(listener.Addr().String(), "conn2")

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
				wrapped := WrapConnection(c, uuid.New().String())
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
	c1 := NewConnection(addr, uuid.New().String())
	c2 := NewConnection(addr, uuid.New().String())
	c3 := NewConnection(addr, uuid.New().String())

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

func TestConnection_ConcurrentUseError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	id := uuid.New().String()
	go func() {
		conn, _ := listener.Accept()
		if conn == nil {
			return
		}
		WrapConnection(conn, id)
		// Keep connection open
		select {}
	}()

	c := NewConnection(listener.Addr().String(), id)
	if c == nil {
		t.Fatal("NewConnection returned nil")
	}

	done := make(chan error, 2)
	
	// Start a blocking Receive
	go func() {
		_, err := c.Receive()
		done <- err
	}()

	// Give Receive time to acquire the lock
	// Then try Send which should fail immediately
	go func() {
		err := c.Send([]byte("test"))
		done <- err
	}()

	// One should succeed (or block), one should fail with errConnectionInUse
	err1 := <-done
	if err1 != nil && err1.Error() != "connection in use: concurrent Send/Receive not allowed" {
		t.Errorf("Expected connection in use error, got: %v", err1)
	}
}

func TestConnection_ChunkedReconnect(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	id := uuid.New().String()
	acceptCount := 0
	
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			acceptCount++
			wrapped := WrapConnection(conn, id)
			
			// Echo back
			data, err := wrapped.Receive()
			if err != nil {
				conn.Close()
				continue
			}
			wrapped.Send(data)
			conn.Close()
		}
	}()

	c := NewConnection(listener.Addr().String(), id)
	if c == nil {
		t.Fatal("NewConnection returned nil")
	}

	// Send large message that requires chunking
	largeData := make([]byte, 200*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err = c.Send(largeData)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	received, err := c.Receive()
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	if len(received) != len(largeData) {
		t.Errorf("Expected %d bytes, got %d", len(largeData), len(received))
	}
}

func TestConnection_MultipleChunks(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	id := uuid.New().String()
	go func() {
		conn, _ := listener.Accept()
		if conn == nil {
			return
		}
		wrapped := WrapConnection(conn, id)
		data, _ := wrapped.Receive()
		wrapped.Send(data)
	}()

	c := NewConnection(listener.Addr().String(), id)
	if c == nil {
		t.Fatal("NewConnection returned nil")
	}

	// Test with exactly 3 chunks (192KB)
	testData := make([]byte, 192*1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	err = c.Send(testData)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	received, err := c.Receive()
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	if len(received) != len(testData) {
		t.Errorf("Expected %d bytes, got %d", len(testData), len(received))
	}

	// Verify data integrity
	for i := range testData {
		if received[i] != testData[i] {
			t.Errorf("Data mismatch at byte %d", i)
			break
		}
	}
}



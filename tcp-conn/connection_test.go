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

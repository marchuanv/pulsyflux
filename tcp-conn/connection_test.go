package tcpconn

import (
	"net"
	"testing"
	"time"
)

func createPipe() (net.Conn, net.Conn) {
	server, client := net.Pipe()
	return server, client
}

func TestConnection_SendReceive(t *testing.T) {
	server, client := createPipe()
	defer client.Close()

	conn := NewConnection(server, 1*time.Second, nil)

	go func() {
		client.Write([]byte("hello"))
	}()

	data, err := conn.Receive()
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	if string(data) != "hello" {
		t.Errorf("Expected 'hello', got '%s'", string(data))
	}

	go func() {
		buf := make([]byte, 1024)
		client.Read(buf)
	}()

	err = conn.Send([]byte("world"))
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestConnection_IdleTimeout(t *testing.T) {
	server, client := createPipe()
	defer client.Close()

	conn := NewConnection(server, 100*time.Millisecond, nil)

	time.Sleep(250 * time.Millisecond)

	err := conn.Send([]byte("test"))
	if err == nil {
		t.Error("Connection should be closed after idle timeout")
	}
}

func TestConnection_ActivityPreventsTimeout(t *testing.T) {
	server, client := createPipe()
	defer client.Close()

	conn := NewConnection(server, 200*time.Millisecond, nil)

	go func() {
		buf := make([]byte, 1024)
		for {
			client.Read(buf)
		}
	}()

	for i := 0; i < 5; i++ {
		time.Sleep(100 * time.Millisecond)
		err := conn.Send([]byte("keep-alive"))
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}
	}

	err := conn.Send([]byte("final"))
	if err != nil {
		t.Error("Connection should still be alive with activity")
	}
}

func TestConnection_ReconnectOnSend(t *testing.T) {
	server1, client1 := createPipe()
	defer client1.Close()

	reconnectCalled := false
	reconnectFn := func() (net.Conn, error) {
		reconnectCalled = true
		server2, client2 := createPipe()
		go func() {
			buf := make([]byte, 1024)
			client2.Read(buf)
		}()
		return server2, nil
	}

	conn := NewConnection(server1, 100*time.Millisecond, reconnectFn)

	time.Sleep(200 * time.Millisecond)

	err := conn.Send([]byte("after reconnect"))
	if err != nil {
		t.Fatalf("Send after reconnect failed: %v", err)
	}

	if !reconnectCalled {
		t.Error("Reconnect function should have been called")
	}
}

func TestConnection_ReconnectOnReceive(t *testing.T) {
	server1, client1 := createPipe()
	defer client1.Close()

	reconnectCalled := false
	reconnectFn := func() (net.Conn, error) {
		reconnectCalled = true
		server2, client2 := createPipe()
		go func() {
			client2.Write([]byte("reconnected"))
		}()
		return server2, nil
	}

	conn := NewConnection(server1, 100*time.Millisecond, reconnectFn)

	time.Sleep(200 * time.Millisecond)

	data, err := conn.Receive()
	if err != nil {
		t.Fatalf("Receive after reconnect failed: %v", err)
	}

	if !reconnectCalled {
		t.Error("Reconnect function should have been called")
	}

	if string(data) != "reconnected" {
		t.Errorf("Expected 'reconnected', got '%s'", string(data))
	}
}

func TestConnection_NoReconnectWithoutFunction(t *testing.T) {
	server, client := createPipe()
	defer client.Close()

	conn := NewConnection(server, 100*time.Millisecond, nil)

	time.Sleep(200 * time.Millisecond)

	err := conn.Send([]byte("test"))
	if err == nil {
		t.Error("Expected error without reconnect function")
	}
}

func TestConnection_ReconnectFailure(t *testing.T) {
	server, client := createPipe()
	defer client.Close()

	reconnectFn := func() (net.Conn, error) {
		return nil, errConnectionDead
	}

	conn := NewConnection(server, 100*time.Millisecond, reconnectFn)

	time.Sleep(200 * time.Millisecond)

	err := conn.Send([]byte("test"))
	if err != errConnectionDead {
		t.Errorf("Expected errConnectionDead, got %v", err)
	}
}

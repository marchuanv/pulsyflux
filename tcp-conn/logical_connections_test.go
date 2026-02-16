package tcpconn

import (
	"github.com/google/uuid"
	"net"
	"testing"
)

func TestLogicalConnections_ServerAndClient(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	// Server: Two logical connections on same physical connection
	go func() {
		conn, _ := listener.Accept()
		if conn == nil {
			return
		}

		session1 := WrapConnection(conn, uuid.MustParse("00000000-0000-0000-0000-000000000001"))
		session2 := WrapConnection(conn, uuid.MustParse("00000000-0000-0000-0000-000000000002"))

		go func() {
			data, _ := session1.Receive()
			session1.Send(append([]byte("session1:"), data...))
		}()

		go func() {
			data, _ := session2.Receive()
			session2.Send(append([]byte("session2:"), data...))
		}()
	}()

	// Client: Two logical connections sharing same physical connection
	client1 := NewConnection(listener.Addr().String(), uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	client2 := NewConnection(listener.Addr().String(), uuid.MustParse("00000000-0000-0000-0000-000000000002"))

	// Send on both logical connections
	client1.Send([]byte("hello"))
	client2.Send([]byte("world"))

	// Receive responses
	resp1, err := client1.Receive()
	if err != nil || string(resp1) != "session1:hello" {
		t.Errorf("client1 expected 'session1:hello', got '%s' err=%v", string(resp1), err)
	}

	resp2, err := client2.Receive()
	if err != nil || string(resp2) != "session2:world" {
		t.Errorf("client2 expected 'session2:world', got '%s' err=%v", string(resp2), err)
	}
}

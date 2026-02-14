package socket

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestClientConnectionRefused(t *testing.T) {
	_, err := NewClient("127.0.0.1:19999", uuid.New())
	if err == nil {
		t.Fatal("Expected connection error, got nil")
	}
}

func TestClientSendAfterClose(t *testing.T) {
	server := NewServer("9091")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	client, err := NewClient("127.0.0.1:9091", uuid.New())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	_, err = client.Send(strings.NewReader("test"), time.Second)
	if err != errClosed {
		t.Errorf("Expected errClosed, got %v", err)
	}
}

func TestClientMultipleClose(t *testing.T) {
	server := NewServer("9095")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	client, err := NewClient("127.0.0.1:9095", uuid.New())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.Close()
	client.Close()
	client.Close()
}

func TestClientEmptyPayload(t *testing.T) {
	server := NewServer("9092")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, err := NewClient("127.0.0.1:9092", channelID)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()

	client2, err := NewClient("127.0.0.1:9092", channelID)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	incoming := make(chan io.Reader, 1)
	outgoing := make(chan io.Reader, 1)
	done := make(chan error, 1)

	go func() {
		done <- client2.Receive(incoming, outgoing, time.Second)
	}()

	go func() {
		req := <-incoming
		data, _ := io.ReadAll(req)
		if len(data) != 0 {
			t.Errorf("Expected empty payload, got %d bytes", len(data))
		}
		outgoing <- strings.NewReader("")
	}()

	response, err := client1.Send(strings.NewReader(""), time.Second)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	data, _ := io.ReadAll(response)
	if len(data) != 0 {
		t.Errorf("Expected empty response, got %d bytes", len(data))
	}

	<-done
}

func TestClientLargePayload(t *testing.T) {
	server := NewServer("9093")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, err := NewClient("127.0.0.1:9093", channelID)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()

	client2, err := NewClient("127.0.0.1:9093", channelID)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	largeData := bytes.Repeat([]byte("X"), 5*1024*1024)

	incoming := make(chan io.Reader, 1)
	outgoing := make(chan io.Reader, 1)
	done := make(chan error, 1)

	go func() {
		done <- client2.Receive(incoming, outgoing, 10*time.Second)
	}()

	go func() {
		req := <-incoming
		data, _ := io.ReadAll(req)
		if len(data) != len(largeData) {
			t.Errorf("Expected %d bytes, got %d", len(largeData), len(data))
		}
		outgoing <- bytes.NewReader(largeData)
	}()

	response, err := client1.Send(bytes.NewReader(largeData), 10*time.Second)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	data, _ := io.ReadAll(response)
	if len(data) != len(largeData) {
		t.Errorf("Expected %d bytes response, got %d", len(largeData), len(data))
	}

	<-done
}

func TestClientZeroTimeout(t *testing.T) {
	server := NewServer("9097")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, err := NewClient("127.0.0.1:9097", channelID)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()

	client2, err := NewClient("127.0.0.1:9097", channelID)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	incoming := make(chan io.Reader, 1)
	outgoing := make(chan io.Reader, 1)
	done := make(chan error, 1)

	go func() {
		done <- client2.Receive(incoming, outgoing, 0)
	}()

	go func() {
		req := <-incoming
		io.ReadAll(req)
		outgoing <- strings.NewReader("ok")
	}()

	response, err := client1.Send(strings.NewReader("test"), 0)
	if err != nil {
		t.Fatalf("Send with zero timeout failed: %v", err)
	}

	data, _ := io.ReadAll(response)
	if string(data) != "ok" {
		t.Errorf("Expected 'ok', got %q", string(data))
	}

	<-done
}

func TestClientReadError(t *testing.T) {
	server := NewServer("9098")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, err := NewClient("127.0.0.1:9098", channelID)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client1.Close()

	client2, err := NewClient("127.0.0.1:9098", channelID)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	incoming := make(chan io.Reader, 1)
	outgoing := make(chan io.Reader, 1)

	go func() {
		client2.Receive(incoming, outgoing, time.Second)
	}()

	errReader := &errorReader{err: errors.New("read error")}

	_, err = client1.Send(errReader, time.Second)
	if err == nil {
		t.Error("Expected read error")
	}
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestClientSelfReceive(t *testing.T) {
	server := NewServer("9099")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client, err := NewClient("127.0.0.1:9099", channelID)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	incoming := make(chan io.Reader, 1)
	outgoing := make(chan io.Reader, 1)

	go func() {
		client.Receive(incoming, outgoing, time.Second)
	}()

	_, err = client.Send(strings.NewReader("self-test"), time.Second)
	if err == nil {
		t.Error("Expected error when client tries concurrent Send/Receive")
	} else if err.Error() != "another operation in progress" {
		t.Errorf("Expected 'another operation in progress', got %v", err)
	}
}

func TestClientSequentialRequests(t *testing.T) {
	server := NewServer("9100")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	channelID := uuid.New()
	client1, err := NewClient("127.0.0.1:9100", channelID)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()

	client2, err := NewClient("127.0.0.1:9100", channelID)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	count := 5
	for i := 0; i < count; i++ {
		incoming := make(chan io.Reader, 1)
		outgoing := make(chan io.Reader, 1)
		done := make(chan error, 1)

		go func() {
			done <- client2.Receive(incoming, outgoing, time.Second)
		}()

		go func() {
			req := <-incoming
			data, _ := io.ReadAll(req)
			outgoing <- strings.NewReader(string(data) + "-echo")
		}()

		msg := "msg" + string(rune('0'+i))
		response, err := client1.Send(strings.NewReader(msg), time.Second)
		if err != nil {
			t.Fatalf("Send %d failed: %v", i, err)
		}

		data, _ := io.ReadAll(response)
		expected := msg + "-echo"
		if string(data) != expected {
			t.Errorf("Request %d: Expected %q, got %q", i, expected, string(data))
		}

		<-done
	}
}

func TestClientBoundaryPayloadSizes(t *testing.T) {
	sizes := []struct {
		name string
		size int
	}{
		{"1byte", 1},
		{"1KB-1", 1023},
		{"1KB", 1024},
		{"1KB+1", 1025},
		{"8KB-1", 8191},
		{"8KB", 8192},
		{"8KB+1", 8193},
		{"maxFrame-1", maxFrameSize - 1},
		{"maxFrame", maxFrameSize},
	}

	for _, tc := range sizes {
		t.Run(tc.name, func(t *testing.T) {
			server := NewServer("9101")
			if err := server.Start(); err != nil {
				t.Fatalf("Failed to start server: %v", err)
			}
			defer server.Stop()

			channelID := uuid.New()
			client1, err := NewClient("127.0.0.1:9101", channelID)
			if err != nil {
				t.Fatalf("Failed to create client1: %v", err)
			}
			defer client1.Close()

			client2, err := NewClient("127.0.0.1:9101", channelID)
			if err != nil {
				t.Fatalf("Failed to create client2: %v", err)
			}
			defer client2.Close()

			data := bytes.Repeat([]byte("A"), tc.size)

			incoming := make(chan io.Reader, 1)
			outgoing := make(chan io.Reader, 1)
			done := make(chan error, 1)

			go func() {
				done <- client2.Receive(incoming, outgoing, 2*time.Second)
			}()

			go func() {
				req := <-incoming
				reqData, _ := io.ReadAll(req)
				if len(reqData) != tc.size {
					t.Errorf("Expected %d bytes, got %d", tc.size, len(reqData))
				}
				outgoing <- bytes.NewReader(reqData)
			}()

			response, err := client1.Send(bytes.NewReader(data), 2*time.Second)
			if err != nil {
				t.Fatalf("Send failed: %v", err)
			}

			respData, _ := io.ReadAll(response)
			if len(respData) != tc.size {
				t.Errorf("Expected %d bytes response, got %d", tc.size, len(respData))
			}

			<-done
		})
	}
}

package socket

import (
	"testing"
)

func TestSocket(test *testing.T) {
	server := NewServer("8080")
	server.Start()
	client := NewClient("8080")
	client.Send("Hello, Server!\n")
	response, err := client.Receive()
	if err != nil {
		test.Errorf("Error receiving response: %v", err)
	}
	if response != "ACK: Hello, Server!\n" {
		test.Errorf("Expected 'Hello, Server!', got '%s'", response)
	}
	client.Close()
	server.Stop()
}

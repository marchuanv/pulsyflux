package connect

import (
	"bytes"
	"net/http"
	"testing"
)

func TestOpenConnectionSendingMessagesToOpenChannel(test *testing.T) {
	conn, err := New("localhost:3000")
	expectedMsg := `{"client_message": "hello, server!"}`
	if err != nil {
		test.Error(err)
	} else {
		go (func() {
			jsonBody := []byte(`{"client_message": "hello, server!"}`)
			bodyReader := bytes.NewReader(jsonBody)
			http.Post("http://localhost:3000", "text/plain", bodyReader)
		})()
		msg, err := conn.GetMessage()
		if err != nil {
			test.Fatal(err)
		}
		if msg != expectedMsg {
			test.Fatal("msg from channel did not match http post request")
		}
	}
}

func TestOpenConnectionSendingMessagesToClosedChannel(test *testing.T) {
	expectedMsg := `{"client_message": "hello, server!"}`
	conn, err := New("localhost:3000")
	if err != nil {
		test.Error(err)
	} else {
		conn.Close()
		go (func() {
			jsonBody := []byte(expectedMsg)
			bodyReader := bytes.NewReader(jsonBody)
			http.Post("http://localhost:3000", "text/plain", bodyReader)
		})()
		msg, err := conn.GetMessage()
		if err == nil {
			test.Fatal("should not receive messages from a closed channel")
		}
		if msg == expectedMsg {
			test.Fatal("should not receive messages from a closed channel")
		}
	}
}

package connect

import (
	"bytes"
	"net/http"
	"testing"
)

func Test(test *testing.T) {
	conn, err := New("localhost:3000")
	if err != nil {
		test.Error(err)
	} else {
		channel := conn.Channel()
		go (func() {
			jsonBody := []byte(`{"client_message": "hello, server!"}`)
			bodyReader := bytes.NewReader(jsonBody)
			http.Post("http://localhost:3000", "text/plain", bodyReader)
		})()
		message := <-channel
		test.Log(message)
	}
}

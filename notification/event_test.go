package notification

import (
	"errors"
	"testing"
	"time"
)

func TestEvent(test *testing.T) {
	expectedData := "Hello World"
	event := New()
	if event == nil {
		test.Fatal()
	}
	event.Subscribe(HTTP_SERVER_CREATE, func(err error, data string) {
		if err != nil {
			test.Fatal(err)
		}
		if data != expectedData {
			test.Fatal(errors.New("data published does not match expected data"))
		}
	})
	time.Sleep(2 * time.Second)
	err := event.Publish(HTTP_SERVER_CREATE, expectedData)
	if err != nil {
		test.Fatal(err)
	}
	time.Sleep(2 * time.Second)
}

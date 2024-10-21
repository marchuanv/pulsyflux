package notification

import (
	"testing"
	"time"
)

func TestEvent(test *testing.T) {
	event := New()
	if event == nil {
		test.Fatal()
	}
	go (func() {
		err := event.Subscribe(HTTP_SERVER_CREATE)
		if err != nil {
			test.Error(err)
		}
	})()
	time.Sleep(5 * time.Second)
	err := event.Publish(HTTP_SERVER_CREATE)
	if err != nil {
		test.Fatal(err)
	}
}

package connect

import (
	notif "pulsyflux/notification"
	"testing"
	"time"
)

func TestServer(test *testing.T) {
	address := "localhost:3000"
	event := newServerEvent()
	event.Subscribe(notif.HTTP_SERVER_STARTED, func(data string, err error) {
		if err != nil {
			event.Publish(notif.CONNECTION_ERROR, err.Error())
		}
	})
	event.Publish(notif.HTTP_SERVER_CREATE, address)
	event.Publish(notif.HTTP_SERVER_START, address)
	time.Sleep(10 * time.Second)
}

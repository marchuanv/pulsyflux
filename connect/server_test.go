package connect

import (
	"pulsyflux/notification"
	"testing"
)

func TestServer(test *testing.T) {
	address := "localhost:3000"
	event := server(address)
	event.Subscribe(notification.HTTP_SERVER_STARTED)
}

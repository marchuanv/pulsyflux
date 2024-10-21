package connect

import (
	"errors"
	"pulsyflux/notification"
	"pulsyflux/util"

	"github.com/google/uuid"
)

type Connection struct {
	Messages chan string
}

var connections = make(map[uuid.UUID]*Connection)

func OpenConnection(address string) (*Connection, error) {
	_, _, err := util.GetHostAndPortFromAddress(address)
	if err != nil {
		return nil, err
	}
	if len(connections) >= 10 {
		return nil, errors.New("connection limit was reached")
	}
	uuid := util.Newv5UUID(address)
	conn := &Connection{}
	existingConn, exists := connections[uuid]
	if exists {
		conn = existingConn
	} else {
		event := server(address)
		go (func() {
			event.Subscribe(notification.HTTP_SERVER_STARTED)
		})()
		go (func() {
			event.Subscribe(notification.HTTP_SERVER_ERROR)
		})()
	}
	return conn, nil
}

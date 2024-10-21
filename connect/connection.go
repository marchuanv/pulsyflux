package connect

import (
	notif "pulsyflux/notification"
)

var connections = make(map[string]*notif.Event)

func NewConnectionEvent(address string) (*notif.Event, error) {
	event, exists := connections[address]
	if !exists {
		event = newServerEvent()
	}
	event.Subscribe(notif.HTTP_SERVER_ERROR, func(addr string, pubErr error) {
		event.Publish(notif.CONNECTION_ERROR, addr)
		connections[addr] = nil
	})
	event.Subscribe(notif.HTTP_SERVER_CREATED, func(data string, pubErr error) {
		event.Publish(notif.HTTP_SERVER_START, address)
	})
	event.Subscribe(notif.HTTP_SERVER_STARTED, func(data string, pubErr error) {
		event.Publish(notif.CONNECTION_OPEN, "")
	})
	event.Publish(notif.HTTP_SERVER_CREATE, address)
	return event, nil
}

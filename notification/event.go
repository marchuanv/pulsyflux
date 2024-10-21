package notification

import (
	"runtime"

	"github.com/google/uuid"
)

type Event struct {
	channel *channel
}

func New() *Event {
	channel, err := new()
	if err != nil {
		panic(err)
	} else {
		event := &Event{channel}
		runtime.SetFinalizer(event, func(e *Event) {
			e.channel.close()
		})
		return event
	}
}

func (e *Event) Publish(notif Notification) error {
	ch, err := get(e, string(notif))
	if err != nil {
		return err
	}
	ch.push(notif)
	return nil
}

func (e *Event) Subscribe(notif Notification) error {
	ch, err := get(e, string(notif))
	if err != nil {
		return err
	}
	_, err = ch.pop()
	if err != nil {
		return err
	}
	return nil
}

func get(e *Event, id string) (*channel, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	var ch *channel
	if e.channel.has(uuid) {
		ch, err = e.channel.get(uuid)
		if err != nil {
			return nil, err
		}
	} else {
		ch, err = e.channel.open(uuid)
		if err != nil {
			return nil, err
		}
	}
	return ch, nil
}

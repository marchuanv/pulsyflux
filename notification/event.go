package notification

import (
	"runtime"

	"github.com/google/uuid"
)

type Event struct {
	channel *Channel
	data    map[Notification]string
}

type RaiseEvent func(err error, data string)

func New() *Event {
	channel, err := new()
	if err != nil {
		panic(err)
	} else {
		event := &Event{
			channel,
			map[Notification]string{},
		}
		runtime.SetFinalizer(event, func(e *Event) {
			e.channel.close()
			for n := range e.data {
				delete(e.data, n)
			}
		})
		return event
	}
}

func (e *Event) Publish(notif Notification, data string) error {
	ch, err := get(e, string(notif))
	if err != nil {
		return err
	}
	ch.push(notif)
	e.data[notif] = data
	return nil
}

func (e *Event) Subscribe(notif Notification, raiseEvent RaiseEvent) {
	go (func() {
		ch, err := get(e, string(notif))
		if err == nil {
			_, err = ch.pop()
			data := e.data[notif]
			raiseEvent(err, data)
		} else {
			raiseEvent(err, "")
		}
	})()
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

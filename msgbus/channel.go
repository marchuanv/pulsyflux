package msgbus

import (
	"errors"
	"maps"
	"pulsyflux/task"
	"runtime"

	"github.com/google/uuid"
)

type Channel struct {
	id       uuid.UUID
	closed   bool
	messages chan string
	channels map[uuid.UUID]*Channel
}

var allChannels = make(map[uuid.UUID]*Channel)

func New(sub MsgSubId) *Channel {
	return task.Do[*Channel, any](func() (*Channel, error) {
		chId := sub.Id()
		channel, exists := allChannels[chId]
		if !exists {
			channel = &Channel{chId, true, make(chan string), make(map[uuid.UUID]*Channel)}
			runtime.SetFinalizer(channel, func(ch *Channel) {
				ch.Close()
			})
		}
		return channel, nil
	})
}

func (ch *Channel) New(sub MsgSubId) *Channel {
	return task.Do[*Channel, any](func() (*Channel, error) {
		chIdRes := sub.Id()
		childCh, exists := ch.channels[chIdRes]
		if !exists {
			childCh = New(sub)
		}
		return childCh, nil
	})
}

func (ch *Channel) Open() bool {
	return task.Do[bool, any](func() (bool, error) {
		ch.closed = false
		return !ch.closed, nil
	})
}

func (ch *Channel) Subscribe() Msg {
	return task.Do[Msg, any](func() (Msg, error) {
		var err error
		var msg Msg
		if ch.closed {
			err = errors.New("channel is closed")
		} else {
			serialisedMsg := <-ch.messages
			msg = NewDeserialisedMessage(serialisedMsg)
		}
		return msg, err
	})
}

func (ch *Channel) Publish(msg Msg) bool {
	return task.Do[bool, any](func() (bool, error) {
		var err error
		if ch.closed {
			err = errors.New("channel is closed")
		} else {
			serialisedMsg := msg.Serialise()
			go (func() {
				ch.messages <- serialisedMsg
			})()
		}
		return true, err
	})
}

func (ch *Channel) Close() bool {
	return task.Do[bool, any](func() (bool, error) {
		var err error
		if len(ch.channels) > 0 {
			for chnKey := range maps.Keys(ch.channels) {
				ch := ch.channels[chnKey]
				ch.Close()
			}
		}
		if !ch.closed {
			close(ch.messages)
			for u := range ch.channels {
				delete(ch.channels, u)
			}
			ch.channels = nil
			ch.messages = nil
			ch.closed = true
		}
		return ch.closed, err
	})
}

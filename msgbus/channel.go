package msgbus

import (
	"errors"
	"maps"
	"runtime"

	"github.com/google/uuid"
)

type Channel struct {
	id       uuid.UUID
	closed   bool
	messages chan string
	channels map[uuid.UUID]*Channel
}

func New(sub MsgSubId) (*Channel, error) {
	chId, err := sub.Id()
	if err != nil {
		return nil, err
	}
	channel := &Channel{
		chId,
		true,
		make(chan string),
		make(map[uuid.UUID]*Channel),
	}
	runtime.SetFinalizer(channel, func(ch *Channel) {
		ch.Close()
	})
	return channel, nil
}

func (ch *Channel) New(sub MsgSubId) (*Channel, error) {
	chId, err := sub.Id()
	if err != nil {
		return nil, err
	}
	childCh, exists := ch.channels[chId]
	if exists {
		return childCh, nil
	}
	childCh, err = New(sub)
	if err != nil {
		return nil, err
	}
	ch.channels[chId] = childCh
	return childCh, nil
}

func (ch *Channel) Open() error {
	ch.closed = false
	return nil
}

func (ch *Channel) Subscribe() (Msg, error) {
	if ch.closed {
		return nil, errors.New("channel is closed")
	}
	serialisedMsg := <-ch.messages
	msg, err := NewDeserialisedMessage(serialisedMsg)
	return msg, err
}

func (ch *Channel) Publish(msg Msg) error {
	if ch.closed {
		return errors.New("channel is closed")
	}
	serialisedMsg, err := msg.Serialise()
	if err == nil {
		go (func() {
			ch.messages <- serialisedMsg
		})()
	}
	return err
}

func (ch *Channel) Close() {
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
}

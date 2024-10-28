package msgbus

import (
	"errors"
	"maps"
	"pulsyflux/util"
	"runtime"

	"github.com/google/uuid"
)

type Channel struct {
	id       uuid.UUID
	closed   bool
	messages chan string
	channels map[uuid.UUID]*Channel
}

func New(sub MsgSubId) *Channel {
	return util.Do(true, func() (*Channel, error) {
		chIdRes := sub.Id()
		channel := &Channel{chIdRes, true, make(chan string), make(map[uuid.UUID]*Channel)}
		runtime.SetFinalizer(channel, func(ch *Channel) {
			ch.Close()
		})
		return channel, nil
	})
}

func (ch *Channel) New(sub MsgSubId) *Channel {
	return util.Do(true, func() (*Channel, error) {
		chIdRes := sub.Id()
		childCh, exists := ch.channels[chIdRes]
		if !exists {
			childCh = New(sub)
		}
		return childCh, nil
	})
}

func (ch *Channel) Open() {
	ch.closed = false
}

func (ch *Channel) Subscribe() Msg {
	return util.Do(true, func() (Msg, error) {
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

func (ch *Channel) Publish(msg Msg) {
	util.Do(true, func() (any, error) {
		var err error
		if ch.closed {
			err = errors.New("channel is closed")
		} else {
			serialisedMsg := msg.Serialise()
			go (func() {
				ch.messages <- serialisedMsg
			})()
		}
		return nil, err
	})
}

func (ch *Channel) Close() {
	util.Do(true, func() (any, error) {
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
		return nil, err
	})
}

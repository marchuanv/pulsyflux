package msgbus

import (
	"errors"
	"fmt"
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
	if len(util.Errors) == 0 {
		chId := sub.Id()
		if len(util.Errors) == 0 {
			channel := &Channel{
				chId,
				true,
				make(chan string),
				make(map[uuid.UUID]*Channel),
			}
			runtime.SetFinalizer(channel, func(ch *Channel) {
				ch.Close()
			})
			return channel
		}
	}
	fmt.Println("there are msgBus errors")
	return nil
}

func (ch *Channel) New(sub MsgSubId) *Channel {
	if len(util.Errors) == 0 {
		chId := sub.Id()
		if len(util.Errors) == 0 {
			childCh, exists := ch.channels[chId]
			if exists {
				return childCh
			}
			childCh = New(sub)
			if len(util.Errors) == 0 {
				ch.channels[chId] = childCh
				return childCh
			}
		}
	}
	fmt.Println("there are msgBus errors")
	return nil
}

func (ch *Channel) Open() {
	if len(util.Errors) == 0 {
		ch.closed = false
	} else {
		fmt.Println("there are msgBus errors")
	}
}

func (ch *Channel) Subscribe() Msg {
	return util.Do(true, func() {
		if ch.closed {
			err := errors.New("channel is closed")
			util.Errors = append(util.Errors, err)
		} else {
			serialisedMsg := <-ch.messages
			util.Do(true, func() {
			})
			msg := NewDeserialisedMessage(serialisedMsg)
			if len(util.Errors) == 0 {
				return msg
			}
		}
	})
	return nil
}

func (ch *Channel) Publish(msg Msg) {
	if len(util.Errors) == 0 {
		if ch.closed {
			err := errors.New("channel is closed")
			util.Errors = append(util.Errors, err)
			return
		}
		serialisedMsg := msg.Serialise()
		if len(util.Errors) == 0 {
			go (func() {
				if len(util.Errors) == 0 {
					ch.messages <- serialisedMsg
				} else {
					fmt.Println("there are msgBus errors")
				}
			})()
		}
	}
	fmt.Println("there are msgBus errors")
}

func (ch *Channel) Close() {
	if len(util.Errors) == 0 {
		if len(ch.channels) > 0 {
			for chnKey := range maps.Keys(ch.channels) {
				ch := ch.channels[chnKey]
				ch.Close()
			}
		}
		if len(util.Errors) == 0 {
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
	}
	fmt.Println("there are msgBus errors")
}

package msgbus

import (
	"errors"
	"maps"
	"pulsyflux/task"
	"runtime"
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	id       uuid.UUID
	closed   bool
	messages chan string
	channels map[uuid.UUID]*Channel
}

const healthMsg = "facf985d-4e85-4c9d-a43b-9684523baace"

var allChannels = make(map[uuid.UUID]*Channel)

func New(sub MsgSubId) *Channel {
	return task.DoNow[*Channel, any](func() *Channel {
		chId := sub.Id()
		channel, exists := allChannels[chId]
		if !exists {
			channel = &Channel{chId, false, make(chan string), make(map[uuid.UUID]*Channel)}
			runtime.SetFinalizer(channel, func(ch *Channel) {
				ch.Close()
			})
			channel.Open()
		}
		return channel
	})
}

func (ch *Channel) New(sub MsgSubId) *Channel {
	return task.DoNow[*Channel, any](func() *Channel {
		chIdRes := sub.Id()
		childCh, exists := ch.channels[chIdRes]
		if !exists {
			childCh = New(sub)
		}
		return childCh
	})
}

func (ch *Channel) Open() bool {
	return task.DoNow[bool, any](func() bool {
		ch.closed = false
		go (func() {
			for !ch.closed {
				time.Sleep(1 * time.Second)
				msg := <-ch.messages
				go (func() {
					if msg != healthMsg {
						ch.messages <- msg
					}
				})()
			}
		})()
		go (func() {
			for !ch.closed {
				ch.messages <- healthMsg
				time.Sleep(1 * time.Second)
			}
		})()
		return !ch.closed
	})
}

func (ch *Channel) Subscribe() Msg {
	return task.DoNow[Msg, any](func() Msg {
		var msg Msg
		if ch.closed {
			err := errors.New("channel is closed")
			panic(err)
		} else {
			serialisedMsg := <-ch.messages
			if serialisedMsg == healthMsg {
				time.Sleep(1 * time.Second)
				return ch.Subscribe()
			} else {
				msg = NewDeserialisedMessage(serialisedMsg)
			}
		}
		return msg
	})
}

func (ch *Channel) Publish(msg Msg) bool {
	return task.DoNow[bool, any](func() bool {
		if ch.closed {
			err := errors.New("channel is closed")
			panic(err)
		} else {
			serialisedMsg := msg.Serialise()
			go (func() {
				ch.messages <- serialisedMsg
			})()
		}
		return true
	})
}

func (ch *Channel) Close() bool {
	return task.DoNow[bool, any](func() bool {
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
		return ch.closed
	})
}

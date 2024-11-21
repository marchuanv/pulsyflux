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
	id         uuid.UUID
	closed     bool
	msgOut     chan string
	msgIn      chan string
	messages   chan string
	channels   map[uuid.UUID]*Channel
	subWaiting bool
}

var allChannels = make(map[uuid.UUID]*Channel)

func New(sub MsgSubId) *Channel {
	return task.DoNow(sub, func(msgSubId MsgSubId) *Channel {
		chId := msgSubId.Id()
		channel, exists := allChannels[chId]
		if !exists {
			channel = &Channel{chId, false, make(chan string), make(chan string), make(chan string), make(map[uuid.UUID]*Channel), false}
			runtime.SetFinalizer(channel, func(ch *Channel) {
				ch.Close()
			})
			channel.Open()
			allChannels[chId] = channel
		}
		return channel
	})
}

func (ch *Channel) New(sub MsgSubId) *Channel {
	return task.DoNow(ch, func(origCh *Channel) *Channel {
		chIdRes := sub.Id()
		childChId, exists := origCh.channels[chIdRes]
		if !exists {
			childChId = New(sub)
			origCh.channels[chIdRes] = childChId
		}
		return childChId
	})
}

func (ch *Channel) Open() bool {
	return task.DoNow(ch, func(origCh *Channel) bool {
		origCh.closed = false
		go (func() {
			for msg := range origCh.msgOut {
				origCh.messages <- msg
			}
		})()
		go (func() {
			for msg := range origCh.messages {
				go (func() {
					time.Sleep(1 * time.Second)
					if origCh.subWaiting {
						origCh.msgIn <- msg
					} else {
						origCh.messages <- msg
					}
				})()
			}
		})()
		return !origCh.closed
	})
}

func (ch *Channel) Subscribe() Msg {
	return task.DoNow(ch, func(origCh *Channel) Msg {
		var msg Msg
		if origCh.closed {
			err := errors.New("channel is closed")
			panic(err)
		} else {
			origCh.subWaiting = true
			serialisedMsg := <-origCh.msgIn
			origCh.subWaiting = false
			msg = NewDeserialisedMessage(serialisedMsg)
		}
		return msg
	})
}

func (ch *Channel) Publish(msg Msg) bool {
	return task.DoNow(ch, func(origCh *Channel) bool {
		if origCh.closed {
			err := errors.New("channel is closed")
			panic(err)
		} else {
			serialisedMsg := msg.Serialise()
			origCh.msgOut <- serialisedMsg
			for childChId := range origCh.channels {
				origChildCh := origCh.channels[childChId]
				origChildCh.msgOut <- serialisedMsg
			}
		}
		return true
	})
}

func (ch *Channel) Close() bool {
	return task.DoNow(ch, func(origCh *Channel) bool {
		if len(origCh.channels) > 0 {
			for chnKey := range maps.Keys(origCh.channels) {
				origCh := origCh.channels[chnKey]
				origCh.Close()
			}
		}
		if !origCh.closed {
			close(origCh.msgIn)
			close(origCh.msgOut)
			close(origCh.messages)
			for u := range origCh.channels {
				delete(origCh.channels, u)
			}
			origCh.channels = nil
			origCh.msgIn = nil
			origCh.msgOut = nil
			origCh.closed = true
		}
		return origCh.closed
	})
}

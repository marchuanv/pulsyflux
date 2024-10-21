package notification

import (
	"errors"
	"maps"

	"github.com/google/uuid"
)

type channel struct {
	channel  chan Notification
	channels map[uuid.UUID]*channel
}

func new() (*channel, error) {
	channel := &channel{
		make(chan Notification),
		make(map[uuid.UUID]*channel),
	}
	return channel, nil
}

func (ch *channel) open(Id uuid.UUID) (*channel, error) {
	if ch.channel == nil {
		ch.channel = make(chan Notification)
		ch.channels = make(map[uuid.UUID]*channel)
	}
	_, exists := ch.channels[Id]
	if exists {
		err := errors.New("channel already created")
		return nil, err
	}
	childCh := &channel{
		make(chan Notification),
		make(map[uuid.UUID]*channel),
	}
	ch.channels[Id] = childCh
	return childCh, nil
}

func (ch *channel) has(Id uuid.UUID) bool {
	_, exists := ch.channels[Id]
	return exists
}

func (ch *channel) get(Id uuid.UUID) (*channel, error) {
	ch, exists := ch.channels[Id]
	if !exists {
		return nil, errors.New("channel not found")
	}
	return ch, nil
}

func (ch *channel) pop() ([]Notification, error) {
	var messages []Notification
	if len(ch.channels) > 0 {
		for childId := range ch.channels {
			child := ch.channels[childId]
			msgs, err := child.pop()
			if err == nil {
				messages = append(messages, msgs...)
			}
		}
	}
	msg := <-ch.channel
	messages = append(messages, msg)
	return messages, nil
}

func (ch *channel) push(msg Notification) {
	if len(ch.channels) > 0 {
		for chnKey := range maps.Keys(ch.channels) {
			ch := ch.channels[chnKey]
			ch.push(msg)
		}
	}
	go func() {
		ch.channel <- msg
	}()
}

func (ch *channel) close() {
	if len(ch.channels) > 0 {
		for chnKey := range maps.Keys(ch.channels) {
			ch := ch.channels[chnKey]
			ch.close()
		}
	}
	close(ch.channel)
	ch.channels = nil
	ch.channel = nil
}

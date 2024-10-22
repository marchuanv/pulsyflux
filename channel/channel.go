package channel

import (
	"errors"
	"maps"

	"github.com/google/uuid"
)

type Channel struct {
	id       uuid.UUID
	channel  chan Message
	channels map[uuid.UUID]*Channel
}

func Open(Id uuid.UUID) (*Channel, error) {
	channel := &Channel{
		Id,
		make(chan Message),
		make(map[uuid.UUID]*Channel),
	}
	return channel, nil
}

func (ch *Channel) Open(Id uuid.UUID) (*Channel, error) {
	relatedCh, exists := ch.channels[Id]
	var err error
	if exists {
		err = errors.New("channel already created")
	} else {
		relatedCh, err = Open(Id)
		if err == nil {
			ch.channels[Id] = relatedCh
		}
	}
	return relatedCh, err
}

func (ch *Channel) Id() uuid.UUID {
	return ch.id
}

func (ch *Channel) Has(Id uuid.UUID) bool {
	_, exists := ch.channels[Id]
	return exists
}

func (ch *Channel) Get(Id uuid.UUID) (*Channel, error) {
	ch, exists := ch.channels[Id]
	if !exists {
		return nil, errors.New("channel not found")
	}
	return ch, nil
}

func (ch *Channel) Pop() ([]Message, error) {
	var messages []Message
	if len(ch.channels) > 0 {
		for childId := range ch.channels {
			child := ch.channels[childId]
			msgs, err := child.Pop()
			if err == nil {
				messages = append(messages, msgs...)
			}
		}
	}
	msg := <-ch.channel
	messages = append(messages, msg)
	return messages, nil
}

func (ch *Channel) Push(msg Message) {
	if len(ch.channels) > 0 {
		for chnKey := range maps.Keys(ch.channels) {
			ch := ch.channels[chnKey]
			ch.Push(msg)
		}
	}
	go func() {
		ch.channel <- msg
	}()
}

func (ch *Channel) Close() {
	if len(ch.channels) > 0 {
		for chnKey := range maps.Keys(ch.channels) {
			ch := ch.channels[chnKey]
			ch.Close()
		}
	}
	close(ch.channel)
	ch.channels = nil
	ch.channel = nil
}

package channel

import (
	"errors"
	"maps"
	"runtime"

	"github.com/google/uuid"
)

type Channel struct {
	id       uuid.UUID
	closed   bool
	channel  chan string
	channels map[uuid.UUID]*Channel
}

func Open(Id uuid.UUID) (*Channel, error) {
	channel := &Channel{
		Id,
		false,
		make(chan string),
		make(map[uuid.UUID]*Channel),
	}
	runtime.SetFinalizer(channel, func(ch *Channel) {
		ch.Close()
	})
	return channel, nil
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

func (ch *Channel) Subscribe(msgSub MsgSub) (Msg, error) {
	uuid, err := uuid.Parse(string(msgSub.id))
	if err != nil {
		return nil, err
	}
	subCh, subChExists := ch.channels[uuid]
	if !subChExists {
		subCh, err = Open(uuid)
		if err == nil {
			ch.channels[uuid] = subCh
		} else {
			return nil, err
		}
	}
	subMsgCh, subMsgChExists := subCh.channels[uuid]
	if !subMsgChExists {
		subMsgCh, err = Open(uuid)
		if err == nil {
			subCh.channels[uuid] = subMsgCh
		} else {
			return nil, err
		}
	}
	msgSubId := <-subCh.channel
	serialisedMsg := <-subMsgCh.channel
	if msgSub.id == MsgSubId(msgSubId) {
		msg, err := NewDeserialisedMessage(serialisedMsg)
		return msg, err
	}
	return nil, err
}

func (ch *Channel) Publish(msgSub MsgSub, msg Msg) error {
	uuid, err := uuid.Parse(string(msgSub.id))
	if err != nil {
		return err
	}
	subCh, subChExists := ch.channels[uuid]
	if !subChExists {
		subCh, err = Open(uuid)
		if err == nil {
			ch.channels[uuid] = subCh
		} else {
			return err
		}
	}
	subMsgCh, subMsgChExists := subCh.channels[uuid]
	if !subMsgChExists {
		subMsgCh, err = Open(uuid)
		if err == nil {
			subCh.channels[uuid] = subMsgCh
		} else {
			return err
		}
	}
	var serialisedMsg string
	serialisedMsg, err = msg.Serialise()
	if err == nil {
		go (func() {
			subCh.channel <- string(msgSub.id)
			subMsgCh.channel <- serialisedMsg
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
		close(ch.channel)
		for u := range ch.channels {
			delete(ch.channels, u)
		}
		ch.channels = nil
		ch.channel = nil
		ch.closed = true
	}
}

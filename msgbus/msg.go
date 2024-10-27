package msgbus

import (
	"encoding/gob"
	"errors"
	"fmt"
	"pulsyflux/util"

	"github.com/google/uuid"
)

type Msg interface {
	GetId() uuid.UUID
	String() string
	Serialise() string
}

type msg struct {
	Id   uuid.UUID
	Text string
}

func NewMessage(text string) Msg {
	if len(util.Errors) == 0 {
		id := uuid.New()
		if len(text) == 0 {
			err := errors.New("the msgText argument is an empty string")
			util.Errors = append(util.Errors, err)
			return nil
		}
		newMsg := msg{id, text}
		gob.Register(newMsg)
		gob.Register(Msg(newMsg))
		return &newMsg
	}
	fmt.Println("there are msgBus errors")
	return nil
}

func NewDeserialisedMessage(serialised string) Msg {
	if len(util.Errors) == 0 {
		var newMsg Msg
		msg, err := util.Deserialise(serialised)
		if err == nil {
			newMsg = msg.(Msg)
			return newMsg
		} else {
			util.Errors = append(util.Errors, err)
		}
	}
	fmt.Println("there are msgBus errors")
	return nil
}

func (m msg) GetId() uuid.UUID {
	return m.Id
}

func (m msg) String() string {
	return m.Text
}

func (m msg) Serialise() string {
	if len(util.Errors) == 0 {
		serialisedStr, err := util.Serialise(Msg(m))
		if err != nil {
			util.Errors = append(util.Errors, err)
		}
		return serialisedStr
	}
	fmt.Println("there are msgBus errors")
	return ""
}

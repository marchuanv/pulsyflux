package msgbus

import (
	"encoding/gob"
	"errors"
	"pulsyflux/util"

	"github.com/google/uuid"
)

type Msg interface {
	GetId() uuid.UUID
	String() string
	Serialise() (string, error)
}

type msg struct {
	Id   uuid.UUID
	Text string
}

func NewMessage(text string) (Msg, error) {
	id := uuid.New()
	if len(text) == 0 {
		return nil, errors.New("the msgText argument is an empty string")
	}
	newMsg := msg{id, text}
	gob.Register(msg{})
	gob.Register(Msg(msg{}))
	return &newMsg, nil
}

func NewDeserialisedMessage(serialised string) (Msg, error) {
	var newMsg Msg
	msg, err := util.Deserialise(serialised)
	newMsg = msg.(Msg)
	return newMsg, err
}

func (m msg) GetId() uuid.UUID {
	return m.Id
}

func (m msg) String() string {
	return m.Text
}

func (m msg) Serialise() (string, error) {
	return util.Serialise(Msg(m))
}

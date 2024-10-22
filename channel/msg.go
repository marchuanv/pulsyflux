package channel

import (
	"encoding/gob"
	"errors"
	"pulsyflux/util"

	"github.com/google/uuid"
)

type Msg interface {
	GetId() uuid.UUID
	GetText() string
	GetAttributes() []MsgAtt
	Serialise() (string, error)
}

type msg struct {
	Id         uuid.UUID
	Text       string
	Attributes []MsgAtt
}

func NewMessage(attributes []MsgAtt, text string) (Msg, error) {
	id := uuid.New()
	if len(attributes) == 0 {
		return nil, errors.New("message requires at least one attribute")
	}
	if len(text) == 0 {
		return nil, errors.New("the msgText argument is an empty string")
	}
	newMsg := msg{id, text, attributes}
	gob.Register(msg{})
	gob.Register(Msg(msg{}))
	gob.Register(msgAtt{})
	gob.Register(MsgAtt(msgAtt{}))
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

func (m msg) GetText() string {
	return m.Text
}

func (m msg) GetAttributes() []MsgAtt {
	return m.Attributes
}

func (m msg) Serialise() (string, error) {
	return util.Serialise(Msg(m))
}

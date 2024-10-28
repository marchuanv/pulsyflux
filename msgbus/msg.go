package msgbus

import (
	"errors"
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
	return util.Do(true, func() (Msg, error) {
		id := uuid.New()
		if len(text) == 0 {
			err := errors.New("the msgText argument is an empty string")
			return nil, err
		}
		newMsg := msg{id, text}
		return newMsg, nil
	})
}

func NewDeserialisedMessage(serialised string) Msg {
	return util.Do(true, func() (Msg, error) {
		desMsg := util.Deserialise[Msg](serialised)
		return desMsg, nil
	})
}

func (m msg) GetId() uuid.UUID {
	return m.Id
}

func (m msg) String() string {
	return m.Text
}

func (m msg) Serialise() string {
	return util.Do(true, func() (string, error) {
		serStr := util.Serialise[Msg](m)
		return serStr, nil
	})
}

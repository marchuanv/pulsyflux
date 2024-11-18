package msgbus

import (
	"errors"
	"pulsyflux/task"
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
	return task.DoNow[Msg, any](func() Msg {
		id := uuid.New()
		if len(text) == 0 {
			err := errors.New("the msgText argument is an empty string")
			if err != nil {
				panic(err)
			}
		}
		return msg{id, text}
	})
}

func NewDeserialisedMessage(serialised string) Msg {
	return task.DoNow[Msg, any](func() Msg {
		desMsg := util.Deserialise[Msg](serialised)
		return desMsg
	})
}

func (m msg) GetId() uuid.UUID {
	return m.Id
}

func (m msg) String() string {
	return m.Text
}

func (m msg) Serialise() string {
	return task.DoNow[string, any](func() string {
		serStr := util.Serialise[Msg](m)
		return serStr
	})
}

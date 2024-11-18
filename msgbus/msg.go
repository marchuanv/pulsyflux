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
	return task.DoNow(text, func(txt string) Msg {
		id := uuid.New()
		if len(txt) == 0 {
			err := errors.New("the msgText argument is an empty string")
			if err != nil {
				panic(err)
			}
		}
		return msg{id, txt}
	})
}

func NewDeserialisedMessage(serialised string) Msg {
	return task.DoNow(serialised, func(ser string) Msg {
		return util.Deserialise[Msg](ser)
	})
}

func (m msg) GetId() uuid.UUID {
	return m.Id
}

func (m msg) String() string {
	return m.Text
}

func (m msg) Serialise() string {
	return task.DoNow(m, func(message msg) string {
		serStr := util.Serialise[Msg](message)
		return serStr
	})
}

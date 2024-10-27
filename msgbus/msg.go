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
	Serialise() *util.Result[string]
}

type msg struct {
	Id   uuid.UUID
	Text string
}

func NewMessage(text string) *util.Result[Msg] {
	return util.Do(true, func() (*util.Result[Msg], error) {
		id := uuid.New()
		if len(text) == 0 {
			err := errors.New("the msgText argument is an empty string")
			return nil, err
		}
		newMsg := msg{id, text}
		gob.Register(newMsg)
		gob.Register(Msg(newMsg))
		result := &util.Result[Msg]{newMsg}
		return result, nil
	})

}

func NewDeserialisedMessage(serialised string) *util.Result[Msg] {
	return util.Do(true, func() (*util.Result[Msg], error) {
		desResult := util.Deserialise[Msg](serialised)
		result := &util.Result[Msg]{desResult.Output}
		return result, nil
	})
}

func (m msg) GetId() uuid.UUID {
	return m.Id
}

func (m msg) String() string {
	return m.Text
}

func (m msg) Serialise() *util.Result[string] {
	return util.Do(true, func() (*util.Result[string], error) {
		serResult := util.Serialise[msg](m)
		result := &util.Result[string]{serResult.Output}
		return result, nil
	})
}

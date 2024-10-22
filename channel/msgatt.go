package channel

import (
	"errors"

	"github.com/google/uuid"
)

type MsgAtt interface {
	GetId() uuid.UUID
	GetName() string
}

type msgAtt struct {
	Id   uuid.UUID
	Name string
}

func NewMsgAtt(name string) (MsgAtt, error) {
	id := uuid.New()
	if len(name) == 0 {
		return nil, errors.New("message attribute name is an empty string")
	}
	msgAtt := msgAtt{id, name}
	return &msgAtt, nil
}

func (ma msgAtt) GetId() uuid.UUID {
	return ma.Id
}

func (ma msgAtt) GetName() string {
	return ma.Name
}

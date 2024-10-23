package channel

import (
	"errors"

	"github.com/google/uuid"
)

type MsgAttId string

type MsgAtt struct{ id MsgAttId }

func NewMsgAtt(Id MsgAttId) MsgAtt {
	uuid, err := uuid.Parse(string(Id))
	if err != nil {
		panic(errors.New("message attribute is not a valid UUID"))
	}
	return MsgAtt{MsgAttId(uuid.String())}
}

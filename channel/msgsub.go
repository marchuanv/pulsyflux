package channel

import (
	"errors"
	"fmt"
	"pulsyflux/util"

	"github.com/google/uuid"
)

type MsgSubId string

type MsgSub struct{ id MsgSubId }

func NewMsgSub(Id MsgSubId, msgAtt ...MsgAtt) MsgSub {
	uuid, err := uuid.Parse(string(Id))
	if err != nil {
		panic(errors.New("message subscription is not a valid UUID"))
	}
	for _, att := range msgAtt {
		if !util.IsValidUUID(string(att.id)) {
			errorMsg := fmt.Sprintf("The \"%s\" attribute is not a valid UUID", att)
			panic(errors.New(errorMsg))
		}
		uuid = util.Newv5UUID(string(att.id) + uuid.String())
	}
	return MsgSub{MsgSubId(uuid.String())}
}

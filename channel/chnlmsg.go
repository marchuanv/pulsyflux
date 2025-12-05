package channel

import (
	"github.com/google/uuid"
)

type chnlMsg func() (any, uuid.UUID)

func newChnlMsg(msg any) *chnlMsg {
	Id := uuid.New()
	_chMsgF := chnlMsg(func() (any, uuid.UUID) {
		return msg, Id
	})
	return &_chMsgF
}

func (chMsg chnlMsg) getMsg() (any, uuid.UUID) {
	return chMsg()
}

func getMsg[T any](chMsg *chnlMsg) (canConvert bool, content T) {
	defer (func() {
		err := recover()
		if err != nil {
			canConvert = false
		}
	})()
	msg, _ := chMsg.getMsg()
	content = msg.(T)
	canConvert = true
	return canConvert, content
}

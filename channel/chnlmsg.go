package channel

import (
	"github.com/google/uuid"
)

type chnlmsg func() (any, uuid.UUID)

func newChnlMsg(data any) *chnlmsg {
	Id := uuid.New()
	_chMsgF := chnlmsg(func() (any, uuid.UUID) {
		return data, Id
	})
	return &_chMsgF
}
func (chMsg chnlmsg) content() (any, uuid.UUID) {
	return chMsg()
}

func convert[T any](msg *chnlmsg) (canConvert bool, converted T) {
	defer (func() {
		err := recover()
		if err != nil {
			canConvert = false
		}
	})()
	content, _ := msg.content()
	return true, content.(T)
}

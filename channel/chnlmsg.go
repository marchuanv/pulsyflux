package channel

import (
	"github.com/google/uuid"
)

type chnlmsg func() (any, uuid.UUID)
type ChannelMsg interface {
	Content() (any, uuid.UUID)
}

func NewChnlMsg(data any) ChannelMsg {
	Id := uuid.New()
	return chnlmsg(func() (any, uuid.UUID) {
		return data, Id
	})
}
func (chMsg chnlmsg) Content() (any, uuid.UUID) {
	return chMsg()
}

func convert[T any](msg ChannelMsg) (canConvert bool, converted T) {
	defer (func() {
		err := recover()
		if err != nil {
			canConvert = false
		}
	})()
	content, _ := msg.Content()
	return true, content.(T)
}

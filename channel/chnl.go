package channel

import (
	"github.com/google/uuid"
)

func NewChl(uuidStr string) ChlId {
	Id, err := uuid.Parse(uuidStr)
	if err != nil {
		panic(err)
	}
	return ChlId(Id)
}

func NewChlSub(uuidStr string) ChlSubId {
	Id, err := uuid.Parse(uuidStr)
	if err != nil {
		panic(err)
	}
	return ChlSubId(Id)
}

func (chId ChlId) String() string {
	uuid := uuid.UUID(chId)
	return uuid.String()
}

func (subId ChlSubId) String() string {
	uuid := uuid.UUID(subId)
	return uuid.String()
}

func (Id ChlId) OpenChnl() {
	Id.register()
}

func (Id ChlId) CloseChnl() {
	Id.unregister()
}

func (Id ChlId) Publish(msg any) {
	nvlp := newChnlMsg(msg)
	for _, chlSubId := range Id.subs().All() {
		chlSubId.callback()(nvlp)
	}
}

func (chlSubId ChlSubId) Subscribe(chlId ChlId, rcvMsg func(msg any)) {
	chlSubId.subscribe(chlId, func(nvlp *chnlMsg) {
		canConv, content := getMsg[any](nvlp)
		if canConv {
			go rcvMsg(content)
		}
	})
}

func (chlSubId ChlSubId) Unsubscribe(chlId ChlId) {
	chlSubId.unsubscribe(chlId)
}

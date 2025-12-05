package channel

import (
	"pulsyflux/sliceext"

	"github.com/google/uuid"
)

type Chl[T any] ChlId
type ChlSub[T any] ChlSubId

var chlIds = sliceext.NewDictionary[string, ChlId]()

func GetChl[T any](uuidStr string) Chl[T] {
	Id, err := uuid.Parse(uuidStr)
	if err != nil {
		panic(err)
	}
	if !chlIds.Has(Id.String()) {
		chlIds.Add(Id.String(), ChlId(Id))
	}
	chlId := chlIds.Get(Id.String())
	return Chl[T](chlId)
}

func GetChlSub[T any](uuidStr string) ChlSub[T] {
	Id, err := uuid.Parse(uuidStr)
	if err != nil {
		panic(err)
	}
	return ChlSub[T](Id)
}

func (ch Chl[T]) String() string {
	uuid := uuid.UUID(ch)
	return uuid.String()
}

func (chSub ChlSub[T]) String() string {
	uuid := uuid.UUID(chSub)
	return uuid.String()
}

func (ch Chl[T]) IsOpen() bool {
	return ChlId(ch).registered()
}

func (ch Chl[T]) Open() {
	ChlId(ch).register()
}

func (ch Chl[T]) Close() {
	ChlId(ch).unregister()
}

func (ch Chl[T]) Publish(msg T) {
	nvlp := newChnlMsg(msg)
	for _, chlSubId := range ChlId(ch).subs().All() {
		chlSubId.callback()(nvlp)
	}
}

func (chSub ChlSub[T]) Subscribe(chl Chl[T], rcvMsg func(msg T)) {
	ChlSubId(chSub).subscribe(ChlId(chl), func(nvlp *chnlMsg) {
		canConv, content := getMsg[T](nvlp)
		if canConv {
			go rcvMsg(content)
		}
	})
}

func (chSub ChlSub[T]) Unsubscribe(chl Chl[T]) {
	ChlSubId(chSub).unsubscribe(ChlId(chl))
}

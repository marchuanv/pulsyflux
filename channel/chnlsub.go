package channel

import (
	"pulsyflux/sliceext"

	"github.com/google/uuid"
)

var rcvContentDic = sliceext.NewDictionary[SubId, func(envlp *chnlEnvlp)]()

type SubId uuid.UUID

func NewSubId(uuidStr string) SubId {
	Id, err := uuid.Parse(uuidStr)
	if err != nil {
		panic(err)
	}
	return SubId(Id)
}

func (sub SubId) rcvEnvlp(envlp *chnlEnvlp) {
	defer (func() {
		rec := recover()
		err, canConv := isError[error](rec)
		if canConv {
			panic(err)
		}
	})()
	cb := rcvContentDic.Get(sub)
	cb(envlp)
}

func (sub SubId) String() string {
	uuid := uuid.UUID(sub)
	return uuid.String()
}

func (sub SubId) callback(rcvEnvlpFunc func(envlp *chnlEnvlp)) {
	rcvContentDic.Add(sub, rcvEnvlpFunc)
}

package channel

import (
	"pulsyflux/sliceext"

	"github.com/google/uuid"
)

var rcvContentDic = sliceext.NewDictionary[subId, func(envlp *chnlEnvlp)]()
var rcvErrDic = sliceext.NewDictionary[subId, func(err error)]()

type subId uuid.UUID

func (sub subId) rcvEnvlp(envlp *chnlEnvlp) {
	cb := rcvContentDic.Get(sub)
	cb(envlp)
}

func (sub subId) rcvError(err error) {
	cb := rcvErrDic.Get(sub)
	cb(err)
}

func (sub subId) callback(rcvEnvlpFunc func(envlp *chnlEnvlp), rcvErrFunc func(err error)) {
	rcvContentDic.Add(sub, rcvEnvlpFunc)
	rcvErrDic.Add(sub, rcvErrFunc)
}

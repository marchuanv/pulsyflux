package channel

import (
	"fmt"
	"pulsyflux/sliceext"

	"github.com/google/uuid"
)

type ChlId uuid.UUID
type ChlSubId uuid.UUID

var chls = sliceext.NewDictionary[ChlId, *sliceext.List[ChlSubId]]()
var subs = sliceext.NewDictionary[ChlSubId, func(msg *chnlMsg)]()

func (chl ChlId) registered() bool {
	return chls.Has(chl)
}

func (chl ChlId) register() {
	defer (func() {
		rec := recover()
		err, canConv := isError[error](rec)
		if canConv {
			panic(err)
		}
	})()
	if chl.registered() {
		msg := fmt.Sprintf("channel Id(%s) is already registered", chl)
		panic(newChnlError(msg))
	}
	chls.Add(chl, sliceext.NewList[ChlSubId]())
}

func (chl ChlId) unregister() {
	defer (func() {
		rec := recover()
		err, canConv := isError[error](rec)
		if canConv {
			panic(err)
		}
	})()
	if !chl.registered() {
		msg := fmt.Sprintf("channel Id(%s) is not registered", chl)
		panic(newChnlError(msg))
	}
	if chls.Get(chl).Len() > 0 {
		msg := fmt.Sprintf("channel Id(%s) has active subscriptions", chl)
		panic(newChnlError(msg))
	}
	chls.Delete(chl)
}

func (chl ChlId) subs() *sliceext.List[ChlSubId] {
	defer (func() {
		rec := recover()
		err, canConv := isError[error](rec)
		if canConv {
			panic(err)
		}
	})()
	if !chl.registered() {
		msg := fmt.Sprintf("channel Id(%s) is not registered", chl)
		panic(newChnlError(msg))
	}
	return chls.Get(chl)
}

func (chlSub ChlSubId) subscribe(chlId ChlId, callback func(msg *chnlMsg)) {
	defer (func() {
		rec := recover()
		err, canConv := isError[error](rec)
		if canConv {
			panic(err)
		}
	})()
	if chlId.subs().Has(chlSub) {
		msg := fmt.Sprintf("subscription Id(%s) is already subscribed to channel Id(%s)", chlSub, chlId)
		panic(newChnlError(msg))
	}
	chls.Get(chlId).Add(chlSub)
	subs.Add(chlSub, callback)
}

func (chlSub ChlSubId) unsubscribe(chlId ChlId) {
	defer (func() {
		rec := recover()
		err, canConv := isError[error](rec)
		if canConv {
			panic(err)
		}
	})()
	if !chlId.subs().Has(chlSub) {
		msg := fmt.Sprintf("subscription Id(%s) not found for channel Id(%s)", chlSub, chlId)
		panic(newChnlError(msg))
	}
	chls.Get(chlId).Delete(chlSub)
	subs.Delete(chlSub)
}

func (chlSub ChlSubId) callback() func(msg *chnlMsg) {
	defer (func() {
		rec := recover()
		err, canConv := isError[error](rec)
		if canConv {
			panic(err)
		}
	})()
	if !subs.Has(chlSub) {
		msg := fmt.Sprintf("subscription Id(%s) not found", chlSub)
		panic(newChnlError(msg))
	}
	return subs.Get(chlSub)
}

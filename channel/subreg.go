package channel

import (
	"fmt"
	"pulsyflux/sliceext"
	"sync"
)

var subs = sliceext.NewDictionary[ChlSubId, *ChlSub]()

func (reg *chlSubRegistry) has(chlId ChlId, chlSubId ChlSubId) bool {
	exists := false
	if subs.Has(chlSubId) {
		reg.chlReg.get(chlId, func(chl *Chl) {
			sub := subs.Get(chlSubId)
			if sub.chlId == chl.Id {
				exists = true
			}
		})
	}
	return exists
}

func (reg *chlSubRegistry) get(chlId ChlId, chlSubId ChlSubId, rcvSubWithLock func(sub *ChlSub)) {
	sub := &ChlSub{}
	sub = nil
	defer (func() {
		rec := recover()
		err, canConv := isError[error](rec)
		if canConv {
			panic(err)
		}
		if sub != nil {
			sub.mu.Unlock()
		}
	})()
	if !reg.has(chlId, chlSubId) {
		msg := fmt.Sprintf("subscription Id(%s) not registered", chlSubId)
		panic(newChnlError(msg))
	}
	sub = subs.Get(chlSubId)
	sub.mu.Lock()
	rcvSubWithLock(sub)
}

func (reg *chlSubRegistry) chlsubs(chlId ChlId, rcvSubWithLock func(sub *ChlSub)) {
	for _, chlSubId := range subs.Keys() {
		reg.get(chlId, chlSubId, func(sub *ChlSub) {
			chlReg.get(chlId, func(chl *Chl) {
				rcvSubWithLock(sub)
			})
		})
	}
}

func (reg *chlSubRegistry) register(chlId ChlId, chlSubId ChlSubId) {
	if reg.has(chlId, chlSubId) {
		msg := fmt.Sprintf("subscription Id(%s) already subscribed to channel Id(%s)", chlSubId, chlId)
		panic(newChnlError(msg))
	}
	subs.Add(chlSubId, &ChlSub{
		chlSubId,
		chlId,
		sync.Mutex{},
		nil,
	})
}

func (reg *chlSubRegistry) unregister(chlId ChlId, chlSubId ChlSubId) {
	chlReg.get(chlId, func(chl *Chl) {
		reg.get(chlId, chlSubId, func(sub *ChlSub) {
			sub.rcvMsg = nil
			subs.Delete(chlSubId)
		})
	})
}

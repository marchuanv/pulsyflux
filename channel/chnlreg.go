package channel

import (
	"fmt"
	"pulsyflux/sliceext"
	"sync"
)

var channels = sliceext.NewDictionary[ChlId, *Chl]()

func (reg *chlRegistry) has(Id ChlId) bool {
	return channels.Has(Id)
}

func (reg *chlRegistry) get(Id ChlId, rcvChlWithLock func(chl *Chl)) {
	ch := &Chl{}
	ch = nil
	defer (func() {
		rec := recover()
		err, canConv := isError[error](rec)
		if canConv {
			panic(err)
		}
		if ch != nil {
			ch.mu.Unlock()
		}
	})()
	if !reg.has(Id) {
		msg := fmt.Sprintf("channel Id(%s) not registered", Id)
		panic(newChnlError(msg))
	}
	ch = channels.Get(Id)
	ch.mu.Lock()
	rcvChlWithLock(ch)
}

func (reg *chlRegistry) register(Id ChlId) {
	if reg.has(Id) {
		msg := fmt.Sprintf("channel Id(%s) already registered", Id)
		panic(newChnlError(msg))
	}
	channels.Add(Id, &Chl{Id, sync.Mutex{}})
}

func (reg *chlRegistry) unregister(Id ChlId) {
	subCount := 0
	reg.chlSubReg.chlsubs(Id, func(sub *ChlSub) {
		subCount += 1
	})
	if subCount > 0 {
		msg := fmt.Sprintf("failed to unregister, channel Id(%s) have active subscriptions", Id)
		panic(newChnlError(msg))
	}
	channels.Delete(Id)
}

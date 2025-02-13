package channel

import (
	"errors"
	"fmt"
	"pulsyflux/sliceext"
	"sync"

	"github.com/google/uuid"
)

type ChnlId uuid.UUID
type Channel struct {
	id   ChnlId
	err  error
	subs *sliceext.Stack[func(envlp *chnlEnvlp)]
	mu   sync.Mutex
}

func NewChnlId(uuidStr string) ChnlId {
	Id, err := uuid.Parse(uuidStr)
	if err != nil {
		panic(err)
	}
	return ChnlId(Id)
}

func (chId ChnlId) String() string {
	uuid := uuid.UUID(chId)
	return uuid.String()
}

var channels = sliceext.NewDictionary[ChnlId, *Channel]()

func HasChnl(Id ChnlId) bool {
	return channels.Has(Id)
}

func OpenChnl(Id ChnlId) {
	if channels.Has(Id) {
		msg := fmt.Sprintf("channel Id(%s) is already open", Id)
		panic(newChnlError(msg))
	} else {
		channels.Add(Id, &Channel{
			Id,
			nil,
			sliceext.NewStack[func(envlp *chnlEnvlp)](),
			sync.Mutex{},
		})
	}
}

func CloseChnl(Id ChnlId) {
	if channels.Has(Id) {
		ch := channels.Get(Id)
		ch.subs.Clear()
		if !channels.Delete(Id) {
			msg := fmt.Sprintf("failed to close channel Id(%s)", Id)
			panic(newChnlError(msg))
		}
	} else {
		msg := fmt.Sprintf("failed to close channel Id(%s) with active subscriptions", Id)
		panic(newChnlError(msg))
	}
}

func Publish[T any](channelId ChnlId, content T) {
	defer (func() {
		rec := recover()
		err, canConv := isError[error](rec)
		if canConv {
			panic(err)
		}
	})()
	if !channels.Has(channelId) {
		msg := fmt.Sprintf("channel Id(%s) is not open. Call OpenChnl() function", channelId)
		panic(newChnlError(msg))
	}
	ch := channels.Get(channelId)
	ch.mu.Lock()
	envlp := newChnlEnvlp(content)
	subsCopy := ch.subs.Clone()
	for subsCopy.Len() > 0 {
		rcvEnvlp := subsCopy.Pop()
		go rcvEnvlp(envlp)
	}
	ch.mu.Unlock()
}

func Subscribe[T any](channelId ChnlId, rcvContent func(content T)) {
	defer (func() {
		rec := recover()
		err, canConv := isError[error](rec)
		if canConv {
			panic(err)
		}
	})()
	if !channels.Has(channelId) {
		msg := fmt.Sprintf("channel Id(%s) is not open. Call OpenChnl() function", channelId)
		panic(errors.New(msg))
	}
	ch := channels.Get(channelId)
	ch.mu.Lock()
	ch.subs.Push(func(envlp *chnlEnvlp) {
		isWantedContent, content := getEnvlpContent[T](envlp)
		if isWantedContent {
			rcvContent(content)
		}
	})
	ch.mu.Unlock()
}

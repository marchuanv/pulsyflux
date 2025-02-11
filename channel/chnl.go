package channel

import (
	"errors"
	"fmt"
	"pulsyflux/sliceext"

	"github.com/google/uuid"
)

type ChnlId uuid.UUID

func (chId ChnlId) String() string {
	uuid := uuid.UUID(chId)
	return uuid.String()
}

var channels = sliceext.NewDictionary[ChnlId, *Channel]()
var channelSubs = sliceext.NewDictionary[ChnlId, *sliceext.List[SubId]]()

type Channel struct {
	id  ChnlId
	err error
}

func OpenChnl(Id ChnlId) {
	if channels.Has(Id) {
		msg := fmt.Sprintf("channel Id(%s) is already open", Id)
		panic(newChnlError(msg))
	} else {
		channels.Add(Id, &Channel{
			Id,
			nil,
		})
	}
}

func CloseChnl(Id ChnlId) {
	if channelSubs.Has(Id) {
		msg := fmt.Sprintf("failed to close channel Id(%s) with active subscriptions", Id)
		panic(newChnlError(msg))
	} else {
		if !channels.Delete(Id) {
			msg := fmt.Sprintf("failed to close channel Id(%s)", Id)
			panic(newChnlError(msg))
		}
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
	if !channelSubs.Has(channelId) {
		msg := fmt.Sprintf("channel Id(%s) does not have subscriptions", channelId)
		panic(newChnlError(msg))
	}
	ch := channels.Get(channelId)
	envlp := newChnlEnvlp(content)
	subs := channelSubs.Get(ch.id)
	for _, sub := range subs.All() {
		go sub.rcvEnvlp(envlp)
	}
}

func Subscribe[T any](sub SubId, channelId ChnlId, rcvContent func(content T)) {
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
	if !channelSubs.Has(channelId) {
		channelSubs.Add(channelId, sliceext.NewList[SubId]())
	}
	subscriptions := channelSubs.Get(channelId)
	if subscriptions.Has(sub) {
		msg := fmt.Sprintf("channel Id(%s) already has subscription %s", channelId, sub)
		panic(errors.New(msg))
	}
	subscriptions.Add(sub)
	sub.callback(func(envlp *chnlEnvlp) {
		isWantedContent, content := getEnvlpContent[T](envlp)
		if isWantedContent {
			rcvContent(content)
		}
	})
}

func Unsubscribe(subId SubId, channelId ChnlId) {
	if !channels.Has(channelId) {
		msg := fmt.Sprintf("channel Id(%s) is not open. Call OpenChnl() function", channelId)
		panic(newChnlError(msg))
	}
	if !channelSubs.Has(channelId) {
		msg := fmt.Sprintf("channel Id(%s) not found for subscriptions", channelId)
		panic(newChnlError(msg))
	}
	subscriptions := channelSubs.Get(channelId)
	if subscriptions.Has(subId) {
		isDeleted := subscriptions.Delete(subId) && channelSubs.Delete(channelId)
		if !isDeleted {
			msg := fmt.Sprintf("failed to unsubscribe from channel Id(%s)", channelId)
			panic(newChnlError(msg))
		}
	} else {
		msg := fmt.Sprintf("failed to unsubscribe from channel Id(%s)", channelId)
		panic(newChnlError(msg))
	}
}

package channel

import (
	"errors"
	"fmt"
	"pulsyflux/sliceext"
	"time"

	"github.com/google/uuid"
)

type chnlId uuid.UUID

var channels = sliceext.NewDictionary[chnlId, *Channel]()
var channelSubs = sliceext.NewDictionary[chnlId, *sliceext.List[subId]]()
var timeoutErrEnvlp = newChnlEnvlp(newChnlError("timeout waiting for channel to receive envelopes"))

type Channel struct {
	Id        chnlId
	timeout   time.Time
	err       error
	envelopes *sliceext.Queue[*chnlEnvlp]
}

func OpenChnl(Id chnlId) {
	if channels.Has(Id) {
		msg := fmt.Sprintf("channel Id(%s) is already open", Id)
		panic(newChnlError(msg))
	} else {
		channels.Add(Id, &Channel{
			Id,
			time.Time{},
			nil,
			sliceext.NewQueue[*chnlEnvlp](),
		})
	}
}

func CloseChnl(Id chnlId) {
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

func Publish[T any](channelId chnlId, content T) {
	defer (func() {
		var chnlErr error
		var err error
		var canConv bool
		rec := recover()
		chnlErr, canConv = isChnlError(rec)
		if canConv {
			panic(chnlErr)
		}
		err, canConv = isError[error](rec)
		if canConv {
			fmt.Print(err)
		}
	})()
	if !channels.Has(channelId) {
		msg := fmt.Sprintf("channel Id(%s) is not open. Call OpenChnl() function", channelId)
		panic(newChnlError(msg))
	}
	ch := channels.Get(channelId)
	ch.envelopes.Enqueue(newChnlEnvlp(content))
	go ch.broadcast()
}

func Subscribe[T any](sub subId, channelId chnlId, rcvContent func(content T), rcvError func(err error)) {
	if !channels.Has(channelId) {
		msg := fmt.Sprintf("channel Id(%s) is not open. Call OpenChnl() function", channelId)
		rcvError(errors.New(msg))
		return
	}
	if !channelSubs.Has(channelId) {
		channelSubs.Add(channelId, sliceext.NewList[subId]())
	}
	subscriptions := channelSubs.Get(channelId)
	if subscriptions.Has(sub) {
		msg := fmt.Sprintf("channel Id(%s) already has subscription %s", channelId, sub)
		rcvError(errors.New(msg))
		return
	}
	subscriptions.Add(sub)
	sub.callback(func(envlp *chnlEnvlp) {
		isWantedContent, content := getEnvlpContent[T](envlp)
		if isWantedContent {
			rcvContent(content)
		}
	}, rcvError)
}

func Unsubscribe(subId subId, channelId chnlId) {
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

func (ch *Channel) broadcast() {
	var envlp *chnlEnvlp
	var wait time.Duration
	if ch.envelopes.Len() == 0 {
		if ch.timeout.IsZero() {
			ch.timeout = time.Now().Add(10 * time.Second)
		}
		if time.Now().UTC().UnixMilli() > ch.timeout.UnixMilli() {
			envlp = timeoutErrEnvlp
		}
		wait = 1000
	} else {
		envlp = ch.envelopes.Dequeue()
		ch.timeout = time.Now().Add(10 * time.Second) //refresh timeout
		wait = 100
	}
	if envlp != nil {
		subs := channelSubs.Get(ch.Id)
		for _, sub := range subs.All() {
			sub.rcvEnvlp(envlp)
		}
	}
	time.Sleep(wait * time.Millisecond)
	if channelSubs.Len() > 0 {
		ch.broadcast()
	}
}

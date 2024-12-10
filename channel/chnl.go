package channel

import (
	"pulsyflux/sliceext"
	"time"
)

type chnl[T any] struct {
	events sliceext.Stack[Msg[ChannelEvent]]
	msgs   sliceext.Queue[Msg[T]]
}

func NewChnl[T any]() Channel[T] {
	ch := &chnl[T]{
		sliceext.NewStack[Msg[ChannelEvent]](),
		sliceext.NewQueue[Msg[T]](),
	}
	ensureGlbChnlEventLookup()
	ch.raiseEvent(glbChnlEventLookup[ChannelWriteReady])
	return ch
}

func (ch *chnl[T]) Receive() Msg[T] {
	defer ch.raiseErrorEvent()
	if ch.hasEvent(glbChnlEventLookup[ChannelReadReady]) {
		msg := ch.msgs.Dequeue()
		ch.raiseEvent(glbChnlEventLookup[ChannelRead])
		return msg
	} else {
		panic("channel is not read ready")
	}
}
func (ch *chnl[T]) OnReceived() {
	ch.waitForEvent(glbChnlEventLookup[ChannelRead])
}

func (ch *chnl[T]) Send(msg Msg[T]) {
	defer ch.raiseErrorEvent()
	if ch.hasEvent(glbChnlEventLookup[ChannelWriteReady]) {
		ch.msgs.Enqueue(msg)
		ch.raiseEvent(glbChnlEventLookup[ChannelReadReady])
	} else {
		panic("channel is not write ready")
	}
}
func (ch *chnl[T]) OnSent() {
	ch.waitForEvent(glbChnlEventLookup[ChannelReadReady])
}

func (ch *chnl[T]) SetReadOnly() {
	defer ch.raiseErrorEvent()
	if ch.hasEventHistory(glbChnlEventLookup[ChannelReadReady]) {
		if ch.msgs.Len() > 0 {
			ch.raiseEvent(glbChnlEventLookup[ChannelReadOnly])
			return
		}
	}
	panic("attempting to make a channel read only when no message was sent")
}

func (ch *chnl[T]) Close() {
	defer ch.raiseErrorEvent()
	if ch.msgs.Len() == 0 {
		ch.raiseEvent(glbChnlEventLookup[ChannelClosed])
	} else {
		panic("attempting to close a channel that has messages")
	}
}

func (ch *chnl[T]) OnClosed() {
	ch.waitForEvent(glbChnlEventLookup[ChannelClosed])
}

func (ch *chnl[T]) waitForEvent(eMsg Msg[ChannelEvent]) {
	if !ch.hasEvent(eMsg) {
		time.Sleep(100 * time.Millisecond)
		ch.waitForEvent(eMsg)
	}
}

func (ch *chnl[T]) hasEvent(eMsg Msg[ChannelEvent]) bool {
	defer ch.raiseErrorEvent()
	poppedMsg := ch.events.ClonePop()
	if poppedMsg != nil {
		msg := poppedMsg.Data()
		eventMsg := eMsg.Data()
		if string(msg) == string(eventMsg) {
			return true
		}
	}
	return false
}

func (ch *chnl[T]) hasEventHistory(eMsg Msg[ChannelEvent]) bool {
	defer ch.raiseErrorEvent()
	poppedMsg := ch.events.ClonePop()
	for poppedMsg != nil {
		msg := poppedMsg.Data()
		eventMsg := eMsg.Data()
		if string(msg) == string(eventMsg) {
			return true
		}
		poppedMsg = ch.events.ClonePop()
	}
	return false
}

func (ch *chnl[T]) raiseErrorEvent() {
	r := recover()
	if r != nil {
		ch.raiseEvent(glbChnlEventLookup[ChannelError])
	}
	ch.events.CloneReset()
}

func (ch *chnl[T]) raiseEvent(eMsg Msg[ChannelEvent]) {
	ch.events.Push(eMsg)
}

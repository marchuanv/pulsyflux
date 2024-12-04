package channel

import (
	"pulsyflux/sliceext"
	"time"
)

type chnl struct {
	events sliceext.Stack[ChannelEvent]
	data   any
}

func NewChnl() Channel {
	ch := &chnl{
		sliceext.NewStack[ChannelEvent](),
		nil,
	}
	ch.RaiseEvent(ChannelWriteReady)
	return ch
}

func (ch *chnl) Read(receiveData func(data any)) {
	defer (func() {
		r := recover()
		if r != nil {
			ch.RaiseEvent(ChannelError)
		}
	})()
	if !ch.HasEvent(ChannelReadReady) {
		panic("channel is not read ready")
	}
	receiveData(ch.data)
	ch.RaiseEvent(ChannelRead)
}

func (ch *chnl) Write(data any) {
	defer (func() {
		r := recover()
		if r != nil {
			ch.RaiseEvent(ChannelError)
		}
	})()
	if !ch.HasEvent(ChannelWriteReady) {
		panic("channel is not write ready")
	}
	ch.data = data
	ch.RaiseEvent(ChannelReadReady)
}

func (ch *chnl) Close() {
	defer (func() {
		r := recover()
		if r != nil {
			ch.RaiseEvent(ChannelError)
		}
	})()
	if ch.HasEvent(ChannelReadReady) {
		panic("attempting to close a channel that is read ready")
	}
}

func (ch *chnl) RaiseEvent(event ChannelEvent) {
	ch.events.Push(event)
}

func (ch *chnl) WaitForEvent(event ChannelEvent) {
	if !ch.HasEvent(event) {
		time.Sleep(100 * time.Millisecond)
		ch.WaitForEvent(event)
	}
}

func (ch *chnl) HasEvent(event ChannelEvent) bool {
	return ch.events.Peek() == event
}

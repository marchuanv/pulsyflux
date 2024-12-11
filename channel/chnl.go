package channel

import (
	"errors"
	"pulsyflux/sliceext"
	"time"
)

type chnl struct {
	state    ChannelState
	err      error
	messages sliceext.Queue[ChannelMsg]
}

func NewChnl() Channel {
	return &chnl{
		newState("Open"),
		nil,
		sliceext.NewQueue[ChannelMsg](),
	}
}

func (ch *chnl) Send(chnlMsg ChannelMsg) {
	ch.raiseErrors()
	ch.messages.Enqueue(chnlMsg)
}

func (ch *chnl) Message() ChannelMsg {
	ch.raiseErrors()
	if ch.messages.Len() == 0 {
		timeout := time.Now().Add(10 * time.Second)
		ch.messages.Enqueue(NewChnlMsg(timeout))
	}
	msg := ch.messages.Dequeue()
	canConvert, timeout := convert[time.Time](msg)
	if canConvert {
		if time.Now().UTC().UnixMilli() > timeout.UnixMilli() {
			ch.err = errors.New("timeout reading message")
			return ch.Message()
		} else {
			ch.messages.Enqueue(NewChnlMsg(timeout))
			time.Sleep(100 * time.Millisecond)
			return ch.Message()
		}
	}
	return msg
}

func (ch *chnl) Close() {
	ch.raiseErrors()
	ch.state = newState("Closed")
}

func (ch *chnl) IsClosed() bool {
	return ch.state.name() == "Closed"
}

func (ch *chnl) hasError() bool {
	return ch.state.name() == "Error"
}

func (ch *chnl) raiseErrors() {
	if ch.IsClosed() {
		ch.err = errors.New("channel is closed")
	}
	if ch.err != nil {
		ch.state = newState("Error")
		panic(ch.err)
	}
}

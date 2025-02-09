package channel

import (
	"errors"
	"pulsyflux/sliceext"
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	state         ChannelState
	err           error
	messages      sliceext.Queue[*chnlmsg]
	subscriptions sliceext.List[uuid.UUID]
}

func NewChnl() *Channel {
	return &Channel{
		newState("Open"),
		nil,
		sliceext.NewQueue[*chnlmsg](),
		sliceext.NewList[uuid.UUID](),
	}
}

func (ch *Channel) Close() {
	ch.raiseErrors()
	ch.state = newState("Closed")
}

func (ch *Channel) isClosed() bool {
	return ch.state.name() == "Closed"
}

func (ch *Channel) send(chnlMsg *chnlmsg) {
	ch.raiseErrors()
	ch.messages.Enqueue(chnlMsg)
}

func (ch *Channel) next() *chnlmsg {
	ch.raiseErrors()
	if ch.messages.Len() == 0 {
		timeout := time.Now().Add(10 * time.Second)
		ch.messages.Enqueue(newChnlMsg(timeout))
	}
	msg := ch.messages.Dequeue()
	canConvert, timeout := convert[time.Time](msg)
	if canConvert {
		if time.Now().UTC().UnixMilli() > timeout.UnixMilli() {
			ch.err = errors.New("timeout dequeing message")
			return ch.next()
		} else if ch.messages.Len() == 0 {
			ch.messages.Enqueue(newChnlMsg(timeout))
			time.Sleep(100 * time.Millisecond)
		}
		return ch.next()
	}
	return msg
}

func (ch *Channel) raiseErrors() {
	if ch.isClosed() {
		ch.err = errors.New("channel is closed")
	}
	if ch.err != nil {
		ch.state = newState("Error")
		panic(ch.err)
	}
}

func Publish[MsgType any](ch *Channel, content MsgType) {
	ch.send(newChnlMsg(content))
}

func Subscribe[MsgType any](subId uuid.UUID, ch *Channel, receive func(msg MsgType)) {
	if !ch.subscriptions.Has(subId) {
		ch.subscriptions.Add(subId)
	}
	go subscribe(subId, ch, receive)
}

func Unsubscribe[MsgType any](subId uuid.UUID, ch *Channel) {
	if ch.subscriptions.Has(subId) {
		ch.subscriptions.Delete(subId)
	}
}

func subscribe[MsgType any](subId uuid.UUID, ch *Channel, receive func(msg MsgType)) {
	if ch.subscriptions.Has(subId) {
		msg := ch.next()
		ch.send(msg)
		canConvert, converted := convert[MsgType](msg)
		if canConvert {
			receive(converted)
		}
		time.Sleep(100 * time.Millisecond)
		subscribe(subId, ch, receive)
	}
}

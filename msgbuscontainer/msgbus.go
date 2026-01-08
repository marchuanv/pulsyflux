package msgbuscontainer

import (
	"context"
	"pulsyflux/shared"
)

type msgBus struct {
	sendAddr   shared.URI
	chlId      shared.Channel
	httpResHdl shared.HttpResHandler
	httpReq    shared.HttpReq
}

func (mb *msgBus) Publish(msg shared.Msg) {
	mb.httpReq.Send(mb.sendAddr, mb.chlId.UUID(), msg)
}

func (mb *msgBus) Subscribe() shared.Msg {
	msg, rcvd := mb.httpResHdl.ReceiveRequest(context.Background())
	if rcvd {
		return msg
	}
	return ""
}

func newMsgBus(
	sendAddr shared.URI,
	chlId shared.Channel,
	httpResHdl shared.HttpResHandler,
	httpReq shared.HttpReq,
) shared.MsgBus {
	return &msgBus{sendAddr, chlId, httpResHdl, httpReq}
}

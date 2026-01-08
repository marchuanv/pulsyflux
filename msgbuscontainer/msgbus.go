package msgbuscontainer

import (
	"context"
	"pulsyflux/contracts"
)

type msgBus struct {
	sendAddr   contracts.URI
	chl        contracts.Channel
	httpResHdl contracts.HttpResHandler
	httpReq    contracts.HttpReq
}

func (mb *msgBus) Publish(msg contracts.Msg) {
	mb.httpReq.Send(mb.sendAddr, mb.chl.UUID(), msg)
}

func (mb *msgBus) Subscribe() contracts.Msg {
	msg, rcvd := mb.httpResHdl.ReceiveRequest(context.Background())
	if rcvd {
		return msg
	}
	return ""
}

func newMsgBus(
	sendAddr contracts.URI,
	chl contracts.Channel,
	httpResHdl contracts.HttpResHandler,
	httpReq contracts.HttpReq,
) contracts.MsgBus {
	return &msgBus{sendAddr, chl, httpResHdl, httpReq}
}

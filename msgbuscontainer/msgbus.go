package msgbuscontainers

import (
	"net/http"
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/httpcontainer"
)

type msgBus[MsgType string] struct {
	chl *contracts.ChannelId[MsgType]
}

func (mb *msgBus[MsgType]) SetChl(chl *contracts.ChannelId[MsgType]) {
	if mb.chl == nil {
		mb.chl = chl // use the pointer passed in
		return
	}
	*(mb.chl) = *chl
	httpResTypeId := contracts.TypeId[contracts.HttpResponse](*chl)
	httpcontainer.HttpResponseConfig(httpResTypeId, http.StatusCreated, "subscribed to channel")
}

func (mb *msgBus[MsgType]) Publish(msg MsgType) {
}

func (mb *msgBus[MsgType]) Subscribe() MsgType {
	httpResTypeId := contracts.TypeId[contracts.HttpResponse](*mb.chl)
	sub := containers.Get[contracts.HttpResponse](httpResTypeId)
	msg := sub.GetMsg()
	return MsgType(msg)
}

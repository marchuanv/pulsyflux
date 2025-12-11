package msgbuscontainers

import (
	"pulsyflux/containers"
	"pulsyflux/contracts"
)

type msgBus[T ~string] struct {
	chl *contracts.ChannelId[T]
}

func (mb *msgBus[T]) SetChl(chl *contracts.ChannelId[T]) {
	if mb.chl == nil {
		mb.chl = chl
	} else {
		*(mb.chl) = *chl
	}
}

func (mb *msgBus[T]) Publish(msg T) {
}

func (mb *msgBus[T]) Subscribe() T {
	httpResTypeId := contracts.TypeId[contracts.HttpResponse](*mb.chl)
	sub := containers.Get[contracts.HttpResponse](httpResTypeId)
	msg := sub.GetMsg()
	return T(msg)
}

package httpcontainer

import (
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
)

type registeredResponses struct {
	list *sliceext.List[contracts.HttpResponse]
}

func (ids *registeredResponses) Init() {
	ids.list = sliceext.NewList[contracts.HttpResponse]()
}

func NewRegResContainer() contracts.Container1[registeredResponses, *registeredResponses] {
	return containers.NewContainer1[registeredResponses, *registeredResponses]()
}

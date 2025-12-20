package httpcontainer

import (
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
)

type registeredResponses struct {
	list *sliceext.List[*httpResHandler]
}

func (ids *registeredResponses) Init() {
	ids.list = sliceext.NewList[*httpResHandler]()
}

func NewRegResContainer() contracts.Container1[registeredResponses, *registeredResponses] {
	return containers.NewContainer1[registeredResponses, *registeredResponses]()
}

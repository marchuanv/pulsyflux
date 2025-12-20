package httpcontainer

import (
	"pulsyflux/containers"
	"pulsyflux/contracts"

	"github.com/google/wire"
)

type httpStatusCode int

func (res httpStatusCode) Init() {
}

func NewHttpStatusContainer() contracts.Container1[httpStatusCode, *httpStatusCode] {
	wire.Build()
	return containers.NewContainer1[httpStatusCode, *httpStatusCode]()
}

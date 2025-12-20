package httpcontainer

import (
	"pulsyflux/containers"
	"pulsyflux/contracts"

	"github.com/google/uuid"
)

type httpMsgId uuid.UUID

func (res httpMsgId) Init() {
}

func NewHttpMsgIdContainer() contracts.Container1[httpMsgId, *httpMsgId] {
	return containers.NewContainer1[httpMsgId, *httpMsgId]()
}

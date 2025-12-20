package httpcontainer

import (
	"pulsyflux/containers"
	"pulsyflux/contracts"
)

type httpStatusMsg string

func (res httpStatusMsg) Init() {
}

func NewHttpStatusMsgContainer() contracts.Container1[httpStatusMsg, *httpStatusMsg] {
	return containers.NewContainer1[httpStatusMsg, *httpStatusMsg]()
}

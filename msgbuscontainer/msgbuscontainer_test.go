package msgbuscontainers

import (
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"testing"
)

func TestMsgBus(test *testing.T) {
	msgBus := containers.Get[contracts.MsgBus[contracts.Msg]](HttpMsgBusId)
	msgBus.Subscribe()
}

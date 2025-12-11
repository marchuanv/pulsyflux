package msgbuscontainers

import (
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/httpcontainer"
	"testing"
)

func TestMsgBus(test *testing.T) {

	uri := containers.Get[contracts.URI](httpcontainer.HttpServerAddressId)
	host := "localhost"
	port := 3000
	path := "publish"

	uri.SetHost(&host)
	uri.SetPort(&port)
	uri.SetPath(&path)

	server := containers.Get[contracts.HttpServer](httpcontainer.HttpServerId)

	go server.Start()

	msgBus := containers.Get[contracts.MsgBus[contracts.Msg]](HttpMsgBusId)
	msg := msgBus.Subscribe()
	msg = "Hello from "
	msgBus.Publish(msg)
}

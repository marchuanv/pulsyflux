package httpcontainers

import (
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"testing"
)

func TestHttpServer(test *testing.T) {

	newHost := "test"

	uri := containers.Get[contracts.URI](HttpServerAddressId)
	test.Log(uri)
	server := containers.Get[contracts.HttpServer](HttpServerId)

	if *server.GetAddress().GetHost() != "localhost" {
		test.Fail()
	}

	server.GetAddress().SetHost(&newHost)

	server2 := containers.Get[contracts.HttpServer](HttpServerId)

	if *server2.GetAddress().GetHost() != newHost {
		test.Fail()
	}

	server2.Start()
}

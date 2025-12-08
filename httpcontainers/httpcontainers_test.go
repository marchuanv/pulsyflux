package httpcontainers

import (
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"testing"
)

func TestHttpServer(test *testing.T) {

	uri := containers.Get[contracts.URI](HttpServerAddressId)
	test.Log(uri)
	server := containers.Get[contracts.HttpServer](HttpServerId)
	server.Start()
	test.Log(server)
	test.Fail()
}

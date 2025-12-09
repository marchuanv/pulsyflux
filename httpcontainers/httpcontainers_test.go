package httpcontainers

import (
	"net/http"
	"net/url"
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"testing"
)

func TestHttpServer(test *testing.T) {

	newHost := "test"

	uri := containers.Get[contracts.URI](HttpServerAddressId)
	host := "localhost"
	port := 3000
	path := "publish"

	uri.SetHost(&host)
	uri.SetPort(&port)
	uri.SetPath(&path)

	server := containers.Get[contracts.HttpServer](HttpServerId)

	if *server.GetAddress().GetHost() != "localhost" {
		test.Fail()
		return
	}

	server.GetAddress().SetHost(&newHost)

	server2 := containers.Get[contracts.HttpServer](HttpServerId)

	if *server2.GetAddress().GetHost() != newHost {
		test.Fail()
	}

	newHost = "localhost"
	server2.GetAddress().SetHost(&newHost)

	url, _ := url.Parse(uri.String())
	responseTypeId := HttpResponseConfig(url, http.StatusOK, "success")
	res := containers.Get[contracts.HttpResponse](responseTypeId)
	if *res.GetSuccessStatusCode() != http.StatusOK {
		test.Fail()
	}

	server3 := containers.Get[contracts.HttpServer](HttpServerId)

	server3.Start()
}

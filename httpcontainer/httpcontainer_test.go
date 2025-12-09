package httpcontainer

import (
	"net/http"
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"testing"

	"github.com/google/uuid"
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

	responseTypeId := contracts.TypeId[contracts.HttpResponse](uuid.NewString())
	HttpResponseConfig(responseTypeId, http.StatusOK, "success")
	res := containers.Get[contracts.HttpResponse](responseTypeId)
	if *res.GetSuccessStatusCode() != http.StatusOK {
		test.Fail()
	}

	server3 := containers.Get[contracts.HttpServer](HttpServerId)

	server3.Start()
}

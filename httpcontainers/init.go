package httpcontainers

import (
	"net/url"
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
	"time"

	"github.com/google/uuid"
)

const (
	HttpRequestHandlerId      contracts.TypeId[httpRequestHandler]                = "02c88068-3a66-4856-b2bf-e2dce244761b"
	httpRequestHandlerNvlpsId contracts.TypeId[sliceext.List[contracts.Envelope]] = "172c2cd8-4869-43a7-aa1a-af341e0b439f"

	HttpServerId               contracts.TypeId[httpServer]   = "fccaca3e-54a4-400d-a9d3-b80a2161c20f"
	HttpServerAddressId        contracts.TypeId[uri]          = "467be880-a887-42dd-b6f8-49591a47c5da"
	httpServerReadTimeoutId    contracts.TypeId[timeDuration] = "49e0575b-a6bc-45d1-aba0-4cf57c48fc06"
	httpServerWriteTimeoutId   contracts.TypeId[timeDuration] = "1958180b-bd2d-4691-8a3f-50a35b5dbaf5"
	httpServerMaxHeaderBytesId contracts.TypeId[int]          = "a22a6899-499f-4779-b66b-75952f4d766a"
)

func uriConfig(uriTypeId contracts.TypeId[uri], protocol *string, host *string, port *int, path *string) {
	protocolTypeId := contracts.TypeId[string](uuid.NewString())
	hostTypeId := contracts.TypeId[string](uuid.NewString())
	pathTypeId := contracts.TypeId[string](uuid.NewString())
	portTypeId := contracts.TypeId[int](uuid.NewString())
	containers.RegisterType(uriTypeId)
	containers.RegisterTypeDependency(uriTypeId, protocolTypeId, "protocol", protocol)
	containers.RegisterTypeDependency(uriTypeId, hostTypeId, "host", host)
	containers.RegisterTypeDependency(uriTypeId, pathTypeId, "path", path)
	containers.RegisterTypeDependency(uriTypeId, portTypeId, "port", port)
}

func init() {

	protocol := "http"
	host := "localhost"
	port := 3000
	path := "unknown"

	uriConfig(HttpServerAddressId, &protocol, &host, &port, &path)

	containers.RegisterType(HttpServerId)
	containers.RegisterTypeDependency(HttpServerId, HttpServerAddressId, "address", nil)
	timeDuration := &timeDuration{duration: 10 * time.Second}
	containers.RegisterTypeDependency(HttpServerId, httpServerReadTimeoutId, "readTimeout", timeDuration)
	containers.RegisterTypeDependency(HttpServerId, httpServerWriteTimeoutId, "writeTimeout", timeDuration)
	maxHeaderBytes := 16 * 1024 //16KB (16 * 1024 bytes)
	containers.RegisterTypeDependency(HttpServerId, httpServerMaxHeaderBytesId, "maxHeaderBytes", &maxHeaderBytes)

	containers.RegisterType(HttpRequestHandlerId)
	containers.RegisterTypeDependency(HttpRequestHandlerId, httpRequestHandlerNvlpsId, "envelopes", nil)

	containers.RegisterTypeDependency(HttpServerId, HttpRequestHandlerId, "handler", nil)
}

func EnvelopeConfig(_url *url.URL) contracts.TypeId[contracts.Envelope] {
	nvlpTypeId := contracts.TypeId[contracts.Envelope](uuid.NewString())
	urlTypeId := contracts.TypeId[url.URL](uuid.NewString())
	envelopeMsgTypeId := contracts.TypeId[any](uuid.NewString())
	containers.RegisterType(nvlpTypeId)
	containers.RegisterTypeDependency(nvlpTypeId, urlTypeId, "url", _url)
	containers.RegisterTypeDependency(nvlpTypeId, envelopeMsgTypeId, "msg", nil)
	return nvlpTypeId
}

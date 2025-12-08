package httpcontainers

import (
	"net/url"
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"

	"github.com/google/uuid"
)

const (
	HttpRequestHandlerId  contracts.TypeId[httpRequestHandler]                     = "02c88068-3a66-4856-b2bf-e2dce244761b"
	HttpConnURIId         contracts.TypeId[uri]                                    = "467be880-a887-42dd-b6f8-49591a47c5da"
	httpRequestHandlersId contracts.TypeId[*sliceext.Stack[contracts.HttpRequest]] = "172c2cd8-4869-43a7-aa1a-af341e0b439f"

	httpServerId               contracts.TypeId[httpServer]   = "fccaca3e-54a4-400d-a9d3-b80a2161c20f"
	httpServerReadTimeoutId    contracts.TypeId[timeDuration] = "49e0575b-a6bc-45d1-aba0-4cf57c48fc06"
	httpServerWriteTimeoutId   contracts.TypeId[timeDuration] = "1958180b-bd2d-4691-8a3f-50a35b5dbaf5"
	httpServerMaxHeaderBytesId contracts.TypeId[int]          = "a22a6899-499f-4779-b66b-75952f4d766a"
)

func uriConfig(uriTypeId contracts.TypeId[uri]) {
	protocolTypeId := contracts.TypeId[string](uuid.NewString())
	hostTypeId := contracts.TypeId[string](uuid.NewString())
	pathTypeId := contracts.TypeId[string](uuid.NewString())
	portTypeId := contracts.TypeId[int](uuid.NewString())
	containers.RegisterType(uriTypeId)
	containers.RegisterTypeDependency(uriTypeId, protocolTypeId, "scheme", nil)
	containers.RegisterTypeDependency(uriTypeId, hostTypeId, "host", nil)
	containers.RegisterTypeDependency(uriTypeId, pathTypeId, "path", nil)
	containers.RegisterTypeDependency(uriTypeId, portTypeId, "port", nil)
}

func init() {

	uriConfig(HttpConnURIId)

	containers.RegisterType(httpServerId)
	containers.RegisterTypeDependency(httpServerId, HttpConnURIId, "address", nil)
	containers.RegisterTypeDependency(httpServerId, httpServerReadTimeoutId, "ReadTimeout", nil)
	containers.RegisterTypeDependency(httpServerId, httpServerWriteTimeoutId, "WriteTimeout", nil)
	containers.RegisterTypeDependency(httpServerId, httpServerMaxHeaderBytesId, "maxHeaderBytes", nil)

	containers.RegisterType(HttpRequestHandlerId)
	containers.RegisterTypeDependency(HttpRequestHandlerId, httpRequestHandlersId, "handlers", nil)

	containers.RegisterTypeDependency(httpServerId, HttpRequestHandlerId, "handler", nil)
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

func URIConfig(_url *url.URL) contracts.TypeId[contracts.URI] {
	uriTypeId := contracts.TypeId[uri](uuid.NewString())
	uriConfig(uriTypeId)
	return uriTypeId
}

// func RegisterConnFactory() factory[contracts.Connection] {
// 	httpConnFactory.register(func(args ...Arg) contracts.Connection {
// 		isUri, uri := argValue[contracts.URI](&args[0])
// 		if isUri {
// 			conn := &httpConnection{}
// 			conn.handlers = sliceext.NewStack[HttpRequest]()
// 			conn.server = &http.Server{
// 				Addr:           uri.String(),
// 				ReadTimeout:    10 * time.Second,
// 				WriteTimeout:   10 * time.Second,
// 				MaxHeaderBytes: 1 << 20,
// 				Handler:        conn,
// 			}
// 			return conn
// 		}
// 		return nil
// 	})
// 	return httpConnFactory
// }

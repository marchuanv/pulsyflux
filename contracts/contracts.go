package contracts

import (
	"net/http"
	"net/url"
	"pulsyflux/sliceext"
	"time"
)

type URI interface {
	GetProtocol() *string
	GetHost() *string
	GetPath() *string
	GetPort() *int

	SetProtocol(protocol *string)
	SetHost(host *string)
	SetPath(path *string)
	SetPort(port *int)

	GetPortStr() string
	GetHostAddress() string
	String() string
}

type Envelope interface {
	GetUrl() *url.URL
	SetUrl(url *url.URL)
	SetMsg(msg *string)
	GetMsg() *string
}

type HttpRequestHandler interface {
	SetHttpResponseIds(httpResponseIds *sliceext.List[TypeId[HttpResponse]])
	ServeHTTP(response http.ResponseWriter, request *http.Request)
}

type HttpResponse interface {
	SetNvlpId(nvlpId *TypeId[Envelope])
	SetNvlpInc(nvlpInc *chan TypeId[Envelope])
	SetNvlpOut(nvlpOut *chan TypeId[Envelope])
	GetSuccessStatusCode() *int
	GetSuccessStatusMsg() *string
	SetSuccessStatusCode(code *int)
	SetSuccessStatusMsg(msg *string)
	GetMsg() any
	SetMsg(msg any)
	Handle(reqHeader http.Header, reqBody string) (reason string, statusCode int, resBody string)
}

type TimeDuration interface {
	GetDuration() time.Duration
	SetDuration(duration time.Duration)
}

type HttpServer interface {
	GetAddress() URI
	SetAddress(addr URI)
	GetReadTimeout() TimeDuration
	SetReadTimeout(duration TimeDuration)
	GetWriteTimeout() TimeDuration
	SetWriteTimeout(duration TimeDuration)
	GetMaxHeaderBytes() int
	SetMaxHeaderBytes(size *int)
	GetHandler() HttpRequestHandler
	SetHandler(handler HttpRequestHandler)
	Start()
	Stop()
}

type ConnectionState interface {
	IsOpen() bool
	IsClosed() bool
}

type TypeId[T any] string

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
	SetMsg(msg any)
	GetMsg() any
}

type HttpRequestHandler interface {
	Receive(nvlpTypeId TypeId[Envelope]) Envelope
	Send(nvlpTypeId TypeId[Envelope])
	GetEnvelopes() *sliceext.List[Envelope]
	SetEnvelopes(handlers *sliceext.List[Envelope])
	ServeHTTP(http.ResponseWriter, *http.Request)
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

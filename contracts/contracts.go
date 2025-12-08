package contracts

import (
	"net/http"
	"net/url"
	"pulsyflux/sliceext"
	"time"
)

type HttpRequest func(response http.ResponseWriter, request *http.Request)

type URI interface {
	GetProtocol() string
	GetHost() string
	GetPath() string
	GetPort() int

	SetProtocol(scheme string)
	SetHost(host string)
	SetPath(path string)
	SetPort(port int)

	PortStr() string
	String() string
}

type Envelope interface {
	GetUrl() url.URL
	SetUrl(url *url.URL)
	SetMsg(msg any)
	GetMsg() any
}

type HttpRequestHandler interface {
	State() ConnectionState
	Close()
	Open()
	Receive(recv func(envelope Envelope))
	Send(envelope Envelope)
	GetHandlers() *sliceext.Stack[HttpRequest]
	SetHandlers(handlers *sliceext.Stack[HttpRequest])
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type TimeDuration interface {
	GetDuration() time.Duration
	SetDuration(duration time.Duration)
}

type HttpServer interface {
	GetAddress() *URI
	SetAddress(addr *URI)
	GetReadTimeout() *TimeDuration
	SetReadTimeout(duration *TimeDuration)
	GetWriteTimeout() *TimeDuration
	SetWriteTimeout(duration *TimeDuration)
	GetMaxHeaderBytes() int
	SetMaxHeaderBytes(size int)
	GetHandler() *HttpRequestHandler
	SetHandler(handler *HttpRequestHandler)
}

type ConnectionState interface {
	IsOpen() bool
	IsClosed() bool
}

type TypeId[T any] string

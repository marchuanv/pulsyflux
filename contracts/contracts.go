package contracts

import (
	"net/http"
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

type HttpRequestHandler interface {
	SetHttpResponseIds(httpResponseIds *sliceext.List[TypeId[HttpResponse]])
	ServeHTTP(response http.ResponseWriter, request *http.Request)
}

type HttpResponse interface {
	SetMsgTypeId(msgTypeId *TypeId[Msg])
	SetIncMsg(incMsg *chan Msg)
	SetOutMsg(outMsg *chan Msg)
	GetSuccessStatusCode() *int
	GetSuccessStatusMsg() *string
	SetSuccessStatusCode(code *int)
	SetSuccessStatusMsg(msg *string)
	GetMsg() Msg
	SetMsg(Msg)
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

type ChannelId[T any] string

type MsgBus[MsgType string] interface {
	SetChl(chl *ChannelId[MsgType])
	Publish(msg MsgType)
	Subscribe() MsgType
}

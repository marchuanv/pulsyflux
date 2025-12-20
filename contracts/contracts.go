package contracts

import (
	"context"
	"net/http"
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
	ServeHTTP(response http.ResponseWriter, request *http.Request)
}

type HttpResponseHandler interface {
	ReceiveRequestMsg(ctx context.Context) (Msg, bool)
	SendResponseMsg(ctx context.Context, msg Msg) bool
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

// Container1: No dependencies
type Container1[T any, PT interface {
	*T
	Init()
}] interface {
	Get() T
}

// Container2: Depends on one other container (A1)
type Container2[T any, A1 any, PA1 interface {
	*A1
	Init()
}, PT interface {
	*T
	Init(Container1[A1, PA1])
}] interface {
	Get() T
}

// Container3: Depends on two other containers (A1, A2)
type Container3[T any, A1, A2 any, PA1 interface {
	*A1
	Init()
}, PA2 interface {
	*A2
	Init()
}, PT interface {
	*T
	Init(Container1[A1, PA1], Container1[A2, PA2])
}] interface {
	Get() T
}

// Container4: Depends on two other containers (A1, A2, A3)
type Container4[T any, A1, A2, A3 any, PA1 interface {
	*A1
	Init()
}, PA2 interface {
	*A2
	Init()
}, PA3 interface {
	*A3
	Init()
}, PT interface {
	*T
	Init(Container1[A1, PA1], Container1[A2, PA2], Container1[A3, PA3])
}] interface {
	Get() T
}

type TypeId[T any] string

type ChannelId[T any] string

type MsgBus[T ~string] interface {
	SetChl(chl *ChannelId[T])
	Publish(msg T)
	Subscribe() T
}

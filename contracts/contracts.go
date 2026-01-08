package contracts

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type URI interface {
	GetPortStr() string
	GetHostAddress() string
	String() string
}

type HttpReqHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type HttpReq interface {
	Send(addr URI, msgId uuid.UUID, content string) (status HttpStatus, resBody string, err error)
}

type HttpResHandler interface {
	ReceiveRequest(ctx context.Context) (Msg, bool)
	RespondToRequest(ctx context.Context, msg Msg) bool
	MsgId() MsgId
}

type TimeDuration interface {
	GetDuration() time.Duration
}

type HttpStatus interface {
	Code() int
	String() string
}

type HttpServer interface {
	GetAddress() URI
	GetResponseHandler(msgId MsgId) HttpResHandler
	Start()
	Stop()
}

type HttpResCollection interface {
	Add(handler HttpResHandler)
}

type ConnectionState interface {
	IsOpen() bool
	IsClosed() bool
}

type ReadTimeDuration TimeDuration

type WriteTimeDuration TimeDuration

type IdleConnTimeoutDuration TimeDuration

type RequestTimeoutDuration TimeDuration

type ResponseTimeoutDuration TimeDuration

type RequestHeadersTimeoutDuration TimeDuration

type TypeId[T any] string

type ChannelId[T any] string

type MsgBus[T ~string] interface {
	SetChl(chl *ChannelId[T])
	Publish(msg T)
	Subscribe() T
}

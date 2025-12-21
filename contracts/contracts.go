package contracts

import (
	"context"
	"net/http"
	"time"
)

type URI interface {
	GetPortStr() string
	GetHostAddress() string
	String() string
}

type HttpReqHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type HttpResHandler interface {
	ReceiveRequestMsg(ctx context.Context) (Msg, bool)
	SendResponseMsg(ctx context.Context, msg Msg) bool
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
	Response(msgId MsgId) HttpResHandler
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

// Container1: No dependencies
type Container1[T any,
	PT interface {
		*T
		Init()
	}] interface {
	Get() T
}

// Container2: Depends on one other container (A1)
type Container2[T any, A1 any,
	PA1 interface {
		*A1
		Init()
	},
	PT interface {
		*T
		Init(Container1[A1, PA1])
	}] interface {
	Get() T
}

// Container3: Depends on two other containers (A1, A2)
type Container3[T any, A1 any, A2 any,
	PA1 interface {
		*A1
		Init()
	},
	PA2 interface {
		*A2
		Init()
	},
	PT interface {
		*T
		Init(Container1[A1, PA1], Container1[A2, PA2])
	}] interface {
	Get() T
}

// Container4: Depends on two other containers (A1, A2, A3)
type Container4[T any, A1 any, A2 any, A3 any,
	PA1 interface {
		*A1
		Init()
	},
	PA2 interface {
		*A2
		Init()
	},
	PA3 interface {
		*A3
		Init()
	},
	PT interface {
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

package shared

import "github.com/google/uuid"

type Channel interface {
	String() string
	UUID() uuid.UUID
	IsNil() bool
}

type MsgBus interface {
	Publish(msg Msg)
	Subscribe() Msg
}

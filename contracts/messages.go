package contracts

import "github.com/google/uuid"

type Msg string

type MsgId interface {
	String() string
	UUID() uuid.UUID
	IsNil() bool
}

type Event struct {
	Id   uuid.UUID
	name string
}

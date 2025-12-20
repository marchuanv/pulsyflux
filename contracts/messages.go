package contracts

import "github.com/google/uuid"

type Msg string

type MsgId uuid.UUID

type Event struct {
	Id   uuid.UUID
	name string
}

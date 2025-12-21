package httpcontainer

import (
	"pulsyflux/contracts"

	"github.com/google/uuid"
)

type httpMsgId uuid.UUID

func (msgId httpMsgId) String() string {
	return uuid.UUID(msgId).String()
}
func (msgId httpMsgId) UUID() uuid.UUID {
	return uuid.UUID(msgId)
}
func (msgId httpMsgId) IsNil() bool {
	return uuid.UUID(msgId) == uuid.Nil
}

func newHttpMsgId(msgUd uuid.UUID) contracts.MsgId {
	return httpMsgId(msgUd)
}

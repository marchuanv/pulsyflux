package subscriptions

import (
	"github.com/google/uuid"
)

type subscId string

func (msgSubId subscId) Id() uuid.UUID {
	id, err := uuid.Parse(string(msgSubId))
	if err != nil {
		panic(err)
	}
	return id
}

package msgbus

import (
	"pulsyflux/task"

	"github.com/google/uuid"
)

type MsgSubId string

func (msgSubId MsgSubId) Id() uuid.UUID {
	return task.DoNow(msgSubId, func(subId MsgSubId) uuid.UUID {
		id, err := uuid.Parse(string(subId))
		if err != nil {
			panic(err)
		}
		return id
	})
}

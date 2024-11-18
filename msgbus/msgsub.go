package msgbus

import (
	"pulsyflux/task"

	"github.com/google/uuid"
)

type MsgSubId string

func (msgSubId MsgSubId) Id() uuid.UUID {
	return task.DoNow[uuid.UUID, any](func() uuid.UUID {
		id, err := uuid.Parse(string(msgSubId))
		if err != nil {
			panic(err)
		}
		return id
	})
}

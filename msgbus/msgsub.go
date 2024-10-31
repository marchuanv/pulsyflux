package msgbus

import (
	"pulsyflux/task"

	"github.com/google/uuid"
)

type MsgSubId string

func (msgSubId MsgSubId) Id() uuid.UUID {
	results, _ := task.Do[uuid.UUID, any](func() (uuid.UUID, error) {
		id, err := uuid.Parse(string(msgSubId))
		if err != nil {
			return id, err
		}
		return id, err
	})
	return results
}

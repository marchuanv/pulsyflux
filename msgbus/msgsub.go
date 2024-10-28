package msgbus

import (
	"pulsyflux/util"

	"github.com/google/uuid"
)

type MsgSubId string

func (msgSubId MsgSubId) Id() uuid.UUID {
	return util.Do(true, func() (uuid.UUID, error) {
		id, err := uuid.Parse(string(msgSubId))
		if err != nil {
			return id, err
		}
		return id, err
	})
}

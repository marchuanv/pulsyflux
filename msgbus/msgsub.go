package msgbus

import (
	"pulsyflux/util"

	"github.com/google/uuid"
)

type MsgSubId string

func (msgSubId MsgSubId) Id() uuid.UUID {
	uuid, err := uuid.Parse(string(msgSubId))
	if err != nil {
		util.Errors = append(util.Errors, err)
	}
	return uuid
}

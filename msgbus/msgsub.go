package msgbus

import (
	"github.com/google/uuid"
)

type MsgSubId string

func (msgSubId MsgSubId) Id() (uuid.UUID, error) {
	uuid, err := uuid.Parse(string(msgSubId))
	if err != nil {
		return uuid, err
	}
	return uuid, nil
}

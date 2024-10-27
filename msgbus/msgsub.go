package msgbus

import (
	"pulsyflux/util"

	"github.com/google/uuid"
)

type MsgSubId string

func (msgSubId MsgSubId) Id() *util.Result[uuid.UUID] {
	return util.Do(true, func() (*util.Result[uuid.UUID], error) {
		id, err := uuid.Parse(string(msgSubId))
		if err != nil {
			return nil, err
		}
		result := &util.Result[uuid.UUID]{id}
		return result, err
	})
}

package msgbuscontainer

import (
	"pulsyflux/shared"

	"github.com/google/uuid"
)

type channel struct {
	id       uuid.UUID
	sendAddr shared.URI
}

func (c *channel) UUID() uuid.UUID {
	return c.id
}

func (c *channel) String() string {
	return "default-channel-string-representation"
}

func (c *channel) IsNil() bool {
	return false
}

func newChannel(
	channnelId uuid.UUID,
	sendAddr shared.URI,
) shared.Channel {
	return &channel{channnelId, sendAddr}
}

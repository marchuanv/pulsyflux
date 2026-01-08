package msgbuscontainer

import (
	"pulsyflux/contracts"

	"github.com/google/uuid"
)

type channel struct{}

func (c *channel) UUID() uuid.UUID {
	return uuid.MustParse("default-channel-uuid")
}

func (c *channel) String() string {
	return "default-channel-string-representation"
}

func (c *channel) IsNil() bool {
	return false
}

func newChannel(
	sendAddr contracts.URI,
) contracts.Channel {
	return &channel{}
}

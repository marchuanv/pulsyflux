package subscriptions

import (
	"errors"
	"pulsyflux/channel"
)

func PublishHttpSchema(chnlId channel.ChnlId, httpSchema HttpSchema) {
	if httpSchema.String() == "" {
		channel.Publish(chnlId, errors.New("invalid http schema"))
	} else {
		channel.Publish(chnlId, httpSchema)
	}
}

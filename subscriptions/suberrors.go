package subscriptions

import (
	"pulsyflux/channel"
)

func SubscribeToErrors(chnlId channel.ChnlId, receive func(err error)) {
	channel.Subscribe(chnlId, func(err error) {
		receive(err)
	})
}

package subscriptions

import (
	"pulsyflux/channel"
)

func PublishHostAddress(chnlId channel.ChnlId) {
	SubscribeToAddress(chnlId, func(addr *Address) {
		var hostAddr HostAddress
		hostAddr = HostAddress(*addr)
		channel.Publish(chnlId, hostAddr)
	})
}

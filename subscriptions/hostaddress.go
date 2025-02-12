package subscriptions

import (
	"pulsyflux/channel"
)

type HostAddress Address

func SubscribeToHostAddress(chnlId channel.ChnlId, receive func(addr *HostAddress)) {
	channel.Subscribe(chnlId, func(addr *HostAddress) {
		receive(addr)
	})
}

func PublishHostAddress(chnlId channel.ChnlId) {
	SubscribeToAddress(chnlId, func(addr *Address) {
		var hostAddr HostAddress
		hostAddr = HostAddress(*addr)
		channel.Publish(chnlId, hostAddr)
	})
}

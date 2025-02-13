package subscriptions

import (
	"net"
	"pulsyflux/channel"
)

func PublishHttpListener(chnlId channel.ChnlId) {
	SubscribeToHostAddress(chnlId, func(addr *HostAddress) {
		_listener, err := net.Listen("tcp", addr.String())
		if err == nil {
			channel.Publish(chnlId, _listener)
		} else {
			channel.Publish(chnlId, err)
		}
	})
}

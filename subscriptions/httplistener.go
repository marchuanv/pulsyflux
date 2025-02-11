package subscriptions

import (
	"net"
	"pulsyflux/channel"
)

func SubscribeToHttpListener(chnlId channel.ChnlId, receive func(listener net.Listener, addr *HostAddress)) {
	SubscribeToHostAddress(chnlId, func(addr *HostAddress) {
		channel.Subscribe(chnlId, func(listener net.Listener) {
			receive(listener, addr)
		})
	})
}

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

package subscriptions

import (
	"net"
	"pulsyflux/channel"
)

var HttpListenerSubscription = channel.NewSubId("20875b6d-84c4-4d34-b39c-4596aa4af9ad")

func SubscribeToHttpListener(chnlId channel.ChnlId, receive func(listener net.Listener, addr *HostAddress)) {
	SubscribeToHostAddress(chnlId, func(addr *HostAddress) {
		channel.Subscribe(HttpListenerSubscription, chnlId, func(listener net.Listener) {
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

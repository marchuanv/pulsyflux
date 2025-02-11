package subscriptions

import (
	"net"
	"net/http"
	"pulsyflux/channel"
)

var HttpClientSubscription = channel.NewSubId("c825a25f-e9e8-487e-80f2-df3aa84c0c5d")

func SubscribeToHttpRequest(chnlId channel.ChnlId, receive func(listener net.Listener, server *http.Server, addr *HostAddress)) {
	SubscribeToHttpListener(chnlId, func(listener net.Listener, addr *HostAddress) {
		channel.Subscribe(HttpServerSubscription, chnlId, func(server *http.Server) {
			receive(listener, server, addr)
		})
	})
}

func PublishHttpRequest(chnlId channel.ChnlId) {
	SubscribeToHttpListener(chnlId, func(listener net.Listener, addr *HostAddress) {

	})
}

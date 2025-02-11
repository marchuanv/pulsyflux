package subscriptions

import (
	"net"
	"net/http"
	"pulsyflux/channel"
)

func SubscribeToHttpRequest(chnlId channel.ChnlId, receive func(listener net.Listener, server *http.Server, addr *HostAddress)) {
	SubscribeToHttpListener(chnlId, func(listener net.Listener, addr *HostAddress) {
		channel.Subscribe(chnlId, func(server *http.Server) {
			receive(listener, server, addr)
		})
	})
}

func PublishHttpRequest(chnlId channel.ChnlId) {
	SubscribeToHttpListener(chnlId, func(listener net.Listener, addr *HostAddress) {

	})
}

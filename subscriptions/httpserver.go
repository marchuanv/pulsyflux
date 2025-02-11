package subscriptions

import (
	"net"
	"net/http"
	"pulsyflux/channel"
	"pulsyflux/util"
	"time"
)

var HttpServerSubscription = channel.NewSubId("ca7bed9f-6697-4fdb-9937-8c5e274525b2")

func SubscribeToHttpServer(chnlId channel.ChnlId, receive func(listener net.Listener, server *http.Server, addr *HostAddress)) {
	SubscribeToHttpListener(chnlId, func(listener net.Listener, addr *HostAddress) {
		channel.Subscribe(HttpServerSubscription, chnlId, func(server *http.Server) {
			receive(listener, server, addr)
		})
	})
}

func SubscribeToHttpServerRequest(chnlId channel.ChnlId, receive func(requestData string)) {
	channel.Subscribe(HttpServerSubscription, chnlId, func(response http.ResponseWriter) {
		channel.Subscribe(HttpServerSubscription, chnlId, func(request *http.Request) {
			requestBody := util.StringFromReader(request.Body)
			response.WriteHeader(http.StatusOK)
			response.WriteHeader(http.StatusInternalServerError)
			receive(requestBody)
		})
	})
}

func PublishHttpServer(chnlId channel.ChnlId) {
	SubscribeToHttpListener(chnlId, func(listener net.Listener, addr *HostAddress) {
		httpServer := &http.Server{
			Addr:           addr.String(),
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		httpServer.Handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			channel.Publish(chnlId, response)
			channel.Publish(chnlId, request)
		})
		channel.Publish(chnlId, httpServer)
		err := httpServer.Serve(listener)
		if err != nil {
			channel.Publish(chnlId, err)
		}
	})
}

func PublishStartHttpServer(chnlId channel.ChnlId) {
	SubscribeToHttpServer(chnlId, func(listener net.Listener, server *http.Server, addr *HostAddress) {
		err := server.Serve(listener)
		if err != nil {
			channel.Publish(chnlId, err)
		}
	})
}

func PublishStopHttpServer(chnlId channel.ChnlId) {
	SubscribeToHttpServer(chnlId, func(listener net.Listener, server *http.Server, addr *HostAddress) {
		err := server.Close()
		if err != nil {
			channel.Publish(chnlId, err)
		}
	})
}

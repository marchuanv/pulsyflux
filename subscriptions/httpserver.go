package subscriptions

import (
	"fmt"
	"net"
	"net/http"
	"pulsyflux/channel"
	"pulsyflux/util"
	"time"

	"github.com/google/uuid"
)

type StopServerEvent uuid.UUID

type StartServerEvent uuid.UUID

type ReceiveRequest func() (data string, auth string, contentType string)

func SubscribeToHttpServer(chnlId channel.ChnlId, receive func(listener net.Listener, server *http.Server, addr *HostAddress)) {
	SubscribeToHttpListener(chnlId, func(listener net.Listener, addr *HostAddress) {
		channel.Subscribe(chnlId, func(server *http.Server) {

			channel.Subscribe(chnlId, func(startServer StartServerEvent) {
				err := server.Serve(listener)
				if err != nil {
					channel.Publish(chnlId, err)
				}
			})

			channel.Subscribe(chnlId, func(stopServer StopServerEvent) {
				err := server.Close()
				if err != nil {
					channel.Publish(chnlId, err)
				}
			})

			channel.Subscribe(chnlId, func(response http.ResponseWriter) {
				channel.Subscribe(chnlId, func(receiveRequest ReceiveRequest) {
					data, auth, contentType := receiveRequest()
					response.WriteHeader(http.StatusOK)
					fmt.Println(auth)
					fmt.Println(contentType)
					fmt.Println(data)
					// response.WriteHeader(http.StatusInternalServerError)
				})
			})

			channel.Subscribe(chnlId, func(request *http.Request) {
				_data := util.StringFromReader(request.Body)
				_contentType := request.Header.Get("content-type")
				_auth := request.Header.Get("Authorization")
				channel.Publish(chnlId, ReceiveRequest(func() (data string, auth string, contentType string) {
					return _data, _auth, _contentType
				}))
			})

			receive(listener, server, addr)
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

func PublishStopHttpServer(chnlId channel.ChnlId) {
	channel.Publish(chnlId, StopServerEvent(uuid.New()))
}

func PublishStartHttpServer(chnlId channel.ChnlId) {
	channel.Publish(chnlId, StartServerEvent(uuid.New()))
}

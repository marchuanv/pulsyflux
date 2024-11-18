package connect

import (
	"fmt"
	"net"
	"net/http"
	"pulsyflux/msgbus"
	"pulsyflux/subscriptions"
	"pulsyflux/task"
	"pulsyflux/util"
	"time"
)

func HttpServerSubscriptions() {
	task.DoNow(func() *msgbus.Channel {
		httpCh := msgbus.New(subscriptions.HTTP)
		startHttpServCh := httpCh.New(subscriptions.START_HTTP_SERVER)
		task.DoLater(func() *util.Address {
			receiveHttpServerAddress := startHttpServCh.New(subscriptions.RECEIVE_HTTP_SERVER_ADDRESS)
			msg := receiveHttpServerAddress.Subscribe()
			addressStr := msg.String()
			return util.NewAddress(addressStr)
		}, func(address *util.Address) {
			listener := task.DoNow(func() net.Listener {
				listener, err := net.Listen("tcp", address.String())
				if err != nil {
					panic(err)
				}
				return listener
			}, func(err error, ch *msgbus.Channel) *msgbus.Channel {
				return startHttpServCh.New(subscriptions.FAILED_TO_LISTEN_ON_HTTP_SERVER_PORT)
			})

			httpServer := http.Server{
				Addr:           address.String(),
				ReadTimeout:    10 * time.Second,
				WriteTimeout:   10 * time.Second,
				MaxHeaderBytes: 1 << 20,
			}

			httpServerResCh := httpCh.New(subscriptions.HTTP_SERVER_RESPONSE)
			httpServer.Handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				task.DoNow(func() any {
					requestBody := util.StringFromReader(request.Body)
					resMsg := msgbus.NewDeserialisedMessage(requestBody)
					httpServerSuccResCh := httpServerResCh.New(subscriptions.HTTP_SERVER_SUCCESS_RESPONSE)
					httpServerSuccResCh.Publish(resMsg)
					response.WriteHeader(http.StatusOK)
					return nil
				}, func(err error, param any) any {
					task.DoNow(func() any {
						httpServerErrResCh := httpServerResCh.New(subscriptions.HTTP_SERVER_ERROR_RESPONSE)
						errorMsg := msgbus.NewMessage(err.Error())
						httpServerErrResCh.Publish(errorMsg)
						response.WriteHeader(http.StatusInternalServerError)
						return nil
					}, func(err error, param any) any {
						response.WriteHeader(http.StatusInternalServerError)
						fmt.Println("MessagePublishFail: could not publish the error message to the channel, loggin the error here: ", err)
						return nil
					})
					return nil
				})
			})

			//STOP SERVER
			task.DoNow(func() any {
				stopHttpServCh := httpCh.New(subscriptions.STOP_HTTP_SERVER)
				stopHttpServCh.Subscribe()
				err := httpServer.Close()
				if err != nil {
					panic(err)
				}
				return nil
			}, func(err error, param *msgbus.Channel) *msgbus.Channel {
				stopHttpServCh := httpCh.New(subscriptions.STOP_HTTP_SERVER)
				return stopHttpServCh.New(subscriptions.FAILED_TO_STOP_HTTP_SERVER)
			})

			//START SERVER
			task.DoNow(func() any {
				err := httpServer.Serve(listener)
				if err != nil {
					panic(err)
				}
				msg := msgbus.NewMessage("http server started")
				httpServStartedCh := httpCh.New(subscriptions.HTTP_SERVER_STARTED)
				httpServStartedCh.Publish(msg)
				return nil
			}, func(err error, params *msgbus.Channel) *msgbus.Channel {
				return startHttpServCh.New(subscriptions.FAILED_TO_START_HTTP_SERVER)
			})

		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			return startHttpServCh.New(subscriptions.INVALID_HTTP_SERVER_ADDRESS)
		})
		return nil
	}, func(err error, errorPub *msgbus.Channel) *msgbus.Channel {
		return task.DoNow[*msgbus.Channel, any](func() *msgbus.Channel {
			errorMsg := msgbus.NewMessage(err.Error())
			errorPub.Publish(errorMsg)
			return nil
		})
	})
}

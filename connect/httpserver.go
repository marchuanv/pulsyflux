package connect

import (
	"fmt"
	"net"
	"net/http"
	"pulsyflux/channel"
	"pulsyflux/msgbus"
	"pulsyflux/subscriptions"
	"pulsyflux/task"
	"pulsyflux/util"
	"time"

	"github.com/google/uuid"
)

func HttpServerSubscriptions(chnlId uuid.UUID) {
	subscriptions.SubscribeToNewHostAddress(chnlId, func(addr *subscriptions.HostAddress) {
		listener, err := net.Listen("tcp", addr.String())
		if err != nil {
			panic(err)
		}
	})
	channel.Subscribe(subscriptions.HTTP, ch, func(msg string) {
		channel.Subscribe(subscriptions.START_HTTP_SERVER.Id(), ch, func(httpServerAddress string) {
			util.NewHostAddress(ch, func(hostAddr *util.HostAddress) {

			})
		})
		task.DoLater(httpCh.New(), func(startHttpServCh *msgbus.Channel) *util.Address {
		}, func(address *util.Address, startHttpServCh *msgbus.Channel) {

			listener := task.DoNow(startHttpServCh, func(startHttpServCh *msgbus.Channel) net.Listener {

				return listener
			}, func(err error, startHttpServCh *msgbus.Channel) *msgbus.Channel {
				return startHttpServCh.New(subscriptions.FAILED_TO_LISTEN_ON_HTTP_SERVER_PORT)
			})

			httpServer := http.Server{
				Addr:           address.String(),
				ReadTimeout:    10 * time.Second,
				WriteTimeout:   10 * time.Second,
				MaxHeaderBytes: 1 << 20,
			}

			httpServer.Handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				task.DoNow(startHttpServCh.New(subscriptions.HTTP_SERVER_RESPONSE), func(httpServerResCh *msgbus.Channel) any {
					requestBody := util.StringFromReader(request.Body)
					resMsg := msgbus.NewDeserialisedMessage(requestBody)
					httpServerSuccResCh := httpServerResCh.New(subscriptions.HTTP_SERVER_SUCCESS_RESPONSE)
					httpServerSuccResCh.Publish(resMsg)
					response.WriteHeader(http.StatusOK)
					return nil
				}, func(err error, httpServerResCh *msgbus.Channel) *msgbus.Channel {
					task.DoNow(httpServerResCh, func(httpServerResCh *msgbus.Channel) any {
						httpServerErrResCh := httpServerResCh.New(subscriptions.HTTP_SERVER_ERROR_RESPONSE)
						errorMsg := msgbus.NewMessage(err.Error())
						httpServerErrResCh.Publish(errorMsg)
						response.WriteHeader(http.StatusInternalServerError)
						return nil
					}, func(err error, httpServerResCh *msgbus.Channel) *msgbus.Channel {
						response.WriteHeader(http.StatusInternalServerError)
						fmt.Println("MessagePublishFail: could not publish the error message to the channel, loggin the error here: ", err)
						return httpServerResCh
					})
					return httpServerResCh
				})
			})

			//STOP SERVER
			task.DoLater(startHttpServCh.New(subscriptions.STOP_HTTP_SERVER), func(stopHttpServCh *msgbus.Channel) bool {
				stopHttpServCh.Subscribe()
				err := httpServer.Close()
				if err != nil {
					panic(err)
				}
				return true
			}, func(isListening bool, startHttpServCh *msgbus.Channel) {

			}, func(err error, stopHttpServCh *msgbus.Channel) *msgbus.Channel {
				return stopHttpServCh.New(subscriptions.FAILED_TO_STOP_HTTP_SERVER)
			})

			//START SERVER
			task.DoLater(startHttpServCh.New(subscriptions.HTTP_SERVER_STARTED), func(httpServStartedCh *msgbus.Channel) bool {
				err := httpServer.Serve(listener)
				if err != nil {
					panic(err)
				}
				return true
			}, func(isStarted bool, httpServStartedCh *msgbus.Channel) {
				msg := msgbus.NewMessage("http server started")
				httpServStartedCh.Publish(msg)
			}, func(err error, httpServStartedCh *msgbus.Channel) *msgbus.Channel {
				return httpServStartedCh.New(subscriptions.FAILED_TO_START_HTTP_SERVER)
			})
		}, func(err error, startHttpServCh *msgbus.Channel) *msgbus.Channel {
			return startHttpServCh.New(subscriptions.INVALID_HTTP_SERVER_ADDRESS)
		})
		return httpCh
	}, func(err error, errorPub *msgbus.Channel) *msgbus.Channel {
		return task.DoNow(errorPub, func(errorPub *msgbus.Channel) *msgbus.Channel {
			errorMsg := msgbus.NewMessage(err.Error())
			errorPub.Publish(errorMsg)
			return errorPub
		})
	})
}

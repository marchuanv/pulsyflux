package connect

import (
	"fmt"
	"net"
	"net/http"
	"pulsyflux/msgbus"
	"pulsyflux/subscriptions"
	"pulsyflux/util"
	"time"
)

var httpCh *msgbus.Channel

func Subscribe() (*msgbus.Channel, error) {
	var err error
	var errorMsg msgbus.Msg
	var msg msgbus.Msg
	var httpServStartedCh *msgbus.Channel
	var startHttpServCh *msgbus.Channel
	var stopHttpServCh *msgbus.Channel
	var failedToStartHttpServCh *msgbus.Channel
	var failedToStopHttpServCh *msgbus.Channel
	var httpServPortUnavCh *msgbus.Channel
	var httpServerResCh *msgbus.Channel
	var httpServerSuccResCh *msgbus.Channel
	var httpServerErrResCh *msgbus.Channel
	if httpCh == nil {
		httpCh = msgbus.New(subscriptions.HTTP)
	}
	startHttpServCh = httpCh.New(subscriptions.START_HTTP_SERVER)
	httpServStartedCh = httpCh.New(subscriptions.HTTP_SERVER_STARTED)
	failedToStartHttpServCh = startHttpServCh.New(subscriptions.FAILED_TO_START_HTTP_SERVER)
	stopHttpServCh = httpCh.New(subscriptions.STOP_HTTP_SERVER)
	failedToStopHttpServCh = stopHttpServCh.New(subscriptions.FAILED_TO_STOP_HTTP_SERVER)
	httpServPortUnavCh = startHttpServCh.New(subscriptions.FAILED_TO_LISTEN_ON_HTTP_SERVER_PORT)
	httpServerResCh = httpCh.New(subscriptions.HTTP_SERVER_RESPONSE)
	httpServerSuccResCh = httpServerResCh.New(subscriptions.HTTP_SERVER_SUCCESS_RESPONSE)
	httpServerErrResCh = httpServerResCh.New(subscriptions.HTTP_SERVER_ERROR_RESPONSE)
	util.Do(true, func() {
		msg = startHttpServCh.Subscribe()
		util.Do(true, func() {
			address := msg.String()
			util.GetHostAndPortFromAddress(address)
			util.Do(true, func() {
				listener, err := net.Listen("tcp", address)
				if err == nil {
					httpServer := http.Server{
						Addr:           address,
						ReadTimeout:    10 * time.Second,
						WriteTimeout:   10 * time.Second,
						MaxHeaderBytes: 1 << 20,
					}
					httpServer.Handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
						var resMsg msgbus.Msg
						var err error
						requestBody, err := util.StringFromReader(request.Body)
						if err == nil {
							response.WriteHeader(http.StatusOK)
							resMsg = msgbus.NewDeserialisedMessage(requestBody)
							util.Do(false, func() {
								httpServerSuccResCh.Publish(resMsg)
							})
						} else {
							response.WriteHeader(http.StatusInternalServerError)
							util.Do(false, func() {
								errorMsg = msgbus.NewMessage(err.Error())
								util.Do(false, func() {
									httpServPortUnavCh.Publish(errorMsg)
								})
							})
						}

						return
						err = httpServerSuccResCh.Publish(resMsg)
						if err != nil {
							errorMsg, err = msgbus.NewMessage(err.Error())
							if err == nil {
								err = httpServerErrResCh.Publish(errorMsg)
								if err != nil {
									fmt.Println("failed to publish error msg to the http server error response channel")
								}
							} else {
								fmt.Println("failed to create an error message")
							}
							response.WriteHeader(http.StatusInternalServerError)
							return
						}
					})
					go (func() {
						msg, err = stopHttpServCh.Subscribe()
						if err != nil {
							errorMsg, err = msgbus.NewMessage(err.Error())
							if err == nil {
								err = failedToStopHttpServCh.Publish(errorMsg)
								if err != nil {
									fmt.Println("failed to publish error msg to the failed to stop http server channel")
								}
							} else {
								fmt.Println("failed to create an error message")
							}
							return
						}
						err = httpServer.Close()
						if err != nil {
							errorMsg, err = msgbus.NewMessage(err.Error())
							if err == nil {
								err = failedToStopHttpServCh.Publish(errorMsg)
								if err != nil {
									fmt.Println("failed to publish error msg to the failed to stop http server channel")
								}
							} else {
								fmt.Println("failed to create an error message")
							}
							return
						}
					})()
					err = httpServer.Serve(listener)
					if err == nil {
						msg, err = msgbus.NewMessage("http server started")
						if err == nil {
							err = httpServStartedCh.Publish(msg)
							if err != nil {
								fmt.Println("failed to publish http server started message on channel")
							}
						} else {
							fmt.Println("failed to create http server started message")
						}
					} else {
						errorMsg, err = msgbus.NewMessage(err.Error())
						if err == nil {
							err = failedToStartHttpServCh.Publish(errorMsg)
							if err != nil {
								fmt.Println("failed to publish error msg to the http server failed to start channel")
							}
						} else {
							fmt.Println("failed to create an error message")
						}
						return
					}
				}
			})
		})
	}, func(err error) {
		util.Do(true, func() {
			errorMsg = msgbus.NewMessage(err.Error())
			util.Do(true, func() {
				httpServPortUnavCh.Publish(errorMsg)
			})
		})
	})
	return httpCh, err
}

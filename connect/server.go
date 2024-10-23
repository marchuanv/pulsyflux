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
		httpCh, err = msgbus.New(subscriptions.HTTP)
		if err != nil {
			return nil, err
		}
	}

	startHttpServCh, err = httpCh.New(subscriptions.START_HTTP_SERVER)
	if err != nil {
		return nil, err
	}

	httpServStartedCh, err = httpCh.New(subscriptions.HTTP_SERVER_STARTED)
	if err != nil {
		return nil, err
	}

	failedToStartHttpServCh, err = startHttpServCh.New(subscriptions.FAILED_TO_START_HTTP_SERVER)
	if err != nil {
		return nil, err
	}

	stopHttpServCh, err = httpCh.New(subscriptions.STOP_HTTP_SERVER)
	if err != nil {
		return nil, err
	}
	failedToStopHttpServCh, err = stopHttpServCh.New(subscriptions.FAILED_TO_STOP_HTTP_SERVER)
	if err != nil {
		return nil, err
	}

	httpServPortUnavCh, err = startHttpServCh.New(subscriptions.FAILED_TO_LISTEN_ON_HTTP_SERVER_PORT)
	if err != nil {
		return nil, err
	}

	httpServerResCh, err = httpCh.New(subscriptions.HTTP_SERVER_RESPONSE)
	if err != nil {
		return nil, err
	}

	httpServerSuccResCh, err = httpServerResCh.New(subscriptions.HTTP_SERVER_SUCCESS_RESPONSE)
	if err != nil {
		return nil, err
	}

	httpServerErrResCh, err = httpServerResCh.New(subscriptions.HTTP_SERVER_ERROR_RESPONSE)
	if err != nil {
		return nil, err
	}

	go (func() {
		msg, err = startHttpServCh.Subscribe()
		if err != nil {
			errorMsg, err = msgbus.NewMessage(err.Error())
			if err == nil {
				err = failedToStartHttpServCh.Publish(errorMsg)
				if err != nil {
					fmt.Println("failed to publish error msg to the failed to start http server channel")
				}
			} else {
				fmt.Println("failed to create an error message")
			}
			return
		}
		address := msg.String()
		_, _, err = util.GetHostAndPortFromAddress(address)
		if err != nil {
			errorMsg, err = msgbus.NewMessage(err.Error())
			if err == nil {
				err = httpServPortUnavCh.Publish(errorMsg)
				if err != nil {
					fmt.Println("failed to publish error msg to the http server port unavailable channel")
				}
			} else {
				fmt.Println("failed to create an error message")
			}
			return
		}
		listener, err := net.Listen("tcp", address)
		if err != nil {
			errorMsg, err = msgbus.NewMessage(err.Error())
			if err == nil {
				err = httpServPortUnavCh.Publish(errorMsg)
				if err != nil {
					fmt.Println("failed to publish error msg to the http server port unavailable channel")
				}
			} else {
				fmt.Println("failed to create an error message")
			}
			return
		}
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
			resMsg, err = msgbus.NewDeserialisedMessage(requestBody)
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
	})()
	return httpCh, err
}

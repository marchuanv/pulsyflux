package connect

import (
	"errors"
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
	task.Do(func() (*msgbus.Channel, error) {

		httpCh := msgbus.New(subscriptions.HTTP)
		startHttpServCh := httpCh.New(subscriptions.START_HTTP_SERVER)

		address, errorPubCh := task.Do[*util.Address, *msgbus.Channel](func() (*util.Address, error) {
			receiveHttpServerAddress := startHttpServCh.New(subscriptions.RECEIVE_HTTP_SERVER_ADDRESS)
			msg := receiveHttpServerAddress.Subscribe()
			addressStr := msg.String()
			address := util.NewAddress(addressStr)
			return address, nil
		}, func(err error, params ...*util.Address) *msgbus.Channel {
			invalidHttpServerAddress := startHttpServCh.New(subscriptions.INVALID_HTTP_SERVER_ADDRESS)
			return invalidHttpServerAddress
		})
		if errorPubCh != nil {
			return errorPubCh, errors.New("failed on server address")
		}

		listener, errorPubCh := task.Do[net.Listener, *msgbus.Channel](func() (net.Listener, error) {
			listener, err := net.Listen("tcp", address.String())
			if err != nil {
				return nil, err
			}
			return listener, nil
		}, func(err error, params ...net.Listener) *msgbus.Channel {
			httpServPortUnavCh := startHttpServCh.New(subscriptions.FAILED_TO_LISTEN_ON_HTTP_SERVER_PORT)
			return httpServPortUnavCh
		})
		if errorPubCh != nil {
			return errorPubCh, errors.New("failed to listen on server address")
		}

		httpServer := http.Server{
			Addr:           address.String(),
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		httpServerResCh := httpCh.New(subscriptions.HTTP_SERVER_RESPONSE)
		httpServer.Handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			task.Do(func() (any, error) {
				requestBody := util.StringFromReader(request.Body)
				resMsg := msgbus.NewDeserialisedMessage(requestBody)
				httpServerSuccResCh := httpServerResCh.New(subscriptions.HTTP_SERVER_SUCCESS_RESPONSE)
				httpServerSuccResCh.Publish(resMsg)
				response.WriteHeader(http.StatusOK)
				return nil, nil
			}, func(err error, params ...any) any {
				task.Do(func() (any, error) {
					httpServerErrResCh := httpServerResCh.New(subscriptions.HTTP_SERVER_ERROR_RESPONSE)
					errorMsg := msgbus.NewMessage(err.Error())
					httpServerErrResCh.Publish(errorMsg)
					response.WriteHeader(http.StatusInternalServerError)
					return nil, nil
				}, func(err error, params ...any) any {
					response.WriteHeader(http.StatusInternalServerError)
					fmt.Println("MessagePublishFail: could not publish the error message to the channel, loggin the error here: ", err)
					return nil
				})
				return nil
			})
		})

		//STOP SERVER
		_, errorPubCh = task.Do[any, *msgbus.Channel](func() (any, error) {
			stopHttpServCh := httpCh.New(subscriptions.STOP_HTTP_SERVER)
			stopHttpServCh.Subscribe()
			err := httpServer.Close()
			return nil, err
		}, func(err error, params ...any) *msgbus.Channel {
			stopHttpServCh := httpCh.New(subscriptions.STOP_HTTP_SERVER)
			failedToStopHttpServCh := stopHttpServCh.New(subscriptions.FAILED_TO_STOP_HTTP_SERVER)
			return failedToStopHttpServCh
		})
		if errorPubCh != nil {
			return errorPubCh, errors.New("failed to stop server on address")
		}

		//START SERVER
		_, errorPubCh = task.Do(func() (any, error) {
			err := httpServer.Serve(listener)
			if err != nil {
				return nil, err
			}
			msg := msgbus.NewMessage("http server started")
			httpServStartedCh := httpCh.New(subscriptions.HTTP_SERVER_STARTED)
			httpServStartedCh.Publish(msg)
			return nil, nil
		}, func(err error, params ...any) *msgbus.Channel {
			failedToStartHttpServCh := startHttpServCh.New(subscriptions.FAILED_TO_START_HTTP_SERVER)
			return failedToStartHttpServCh
		})
		if errorPubCh != nil {
			return errorPubCh, errors.New("failed to start server on address")
		}

		return errorPubCh, nil
	}, func(err error, errorPubs ...*msgbus.Channel) any {
		task.Do(func() (any, error) {
			errorMsg := msgbus.NewMessage(err.Error())
			for len(errorPubs) > 0 {
				publisher := errorPubs[len(errorPubs)-1]
				publisher.Publish(errorMsg)
			}
			return nil, nil
		}, func(err error, params ...any) any {
			fmt.Println("MessagePublishFail: could not publish the error message to the channel, logging the error here: ", err)
			return nil
		})
		return nil
	})
}

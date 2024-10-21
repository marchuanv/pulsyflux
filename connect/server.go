package connect

import (
	"net"
	"net/http"
	notif "pulsyflux/notification"
	"pulsyflux/util"
	"time"

	"github.com/google/uuid"
)

func newServerEvent() *notif.Event {
	event := notif.New()
	event.Subscribe(notif.HTTP_SERVER_CREATE, func(address string, pubErr error) {
		_, _, err := util.GetHostAndPortFromAddress(address)
		if err != nil {
			event.Publish(notif.HTTP_SERVER_ERROR, err.Error())
			return
		}
		listener, err := net.Listen("tcp", address)
		pingId := uuid.NewString()
		pingPath := "/" + pingId
		if err == nil {
			httpServer := http.Server{
				Addr:           address,
				ReadTimeout:    10 * time.Second,
				WriteTimeout:   10 * time.Second,
				MaxHeaderBytes: 1 << 20,
			}
			httpServer.Handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.URL.Path == pingPath {
					response.WriteHeader(http.StatusOK)
				} else {
					requestBody, err := util.StringFromReader(request.Body)
					if err == nil {
						response.WriteHeader(http.StatusOK)
						event.Publish(notif.HTTP_SERVER_RESPONSE_RECEVIED, requestBody)
					} else {
						response.WriteHeader(http.StatusInternalServerError)
						event.Publish(notif.HTTP_SERVER_RESPONSE_ERROR, err.Error())
					}
				}
			})
			event.Subscribe(notif.HTTP_SERVER_START, func(data string, pubErr error) {
				if pubErr == nil {
					err := httpServer.Serve(listener)
					if err == nil {
						event.Publish(notif.HTTP_SERVER_STOPPED, "")
					} else {
						event.Publish(notif.HTTP_SERVER_ERROR, err.Error())
					}
				} else {
					event.Publish(notif.HTTP_SERVER_ERROR, pubErr.Error())
				}
			})
			event.Subscribe(notif.HTTP_SERVER_STOP, func(data string, pubErr error) {
				if pubErr == nil {
					servStopErr := httpServer.Close()
					if servStopErr != nil {
						event.Publish(notif.HTTP_SERVER_ERROR, servStopErr.Error())
					}
				} else {
					event.Publish(notif.HTTP_SERVER_ERROR, pubErr.Error())
				}
			})
		}
		_, err = Send(HTTPSchema, HttpGET, address, pingPath, "")
		if err == nil {
			event.Publish(notif.HTTP_SERVER_STARTED, "")
		} else {
			event.Publish(notif.HTTP_SERVER_ERROR, err.Error())
		}
	})
	return event
}

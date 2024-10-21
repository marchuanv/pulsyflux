package connect

import (
	"net"
	"net/http"
	notif "pulsyflux/notification"
	"pulsyflux/util"
	"time"

	"github.com/google/uuid"
)

func server(address string) *notif.Event {
	event := notif.New()
	_, _, err := util.GetHostAndPortFromAddress(address)
	event.Subscribe(notif.HTTP_SERVER_CREATE, func(pubErr error) {
		if err != nil {
			event.Publish(notif.HTTP_SERVER_ERROR)
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
						event.Publish(notif.HTTP_SERVER_RESPONSE_RECEVIED)
					} else {
						response.WriteHeader(http.StatusInternalServerError)
						event.Publish(notif.HTTP_SERVER_RESPONSE_ERROR)
					}
				}
			})
			event.Subscribe(notif.HTTP_SERVER_START, func(pubErr error) {
				if pubErr == nil {
					err = httpServer.Serve(listener)
					if err == nil {
						event.Publish(notif.HTTP_SERVER_STOPPED)
					} else {
						event.Publish(notif.HTTP_SERVER_ERROR)
					}
				} else {
					event.Publish(notif.HTTP_SERVER_ERROR)
				}
			})
			event.Subscribe(notif.HTTP_SERVER_STOP, func(pubErr error) {
				if pubErr == nil {
					servStopErr := httpServer.Close()
					if servStopErr != nil {
						event.Publish(notif.HTTP_SERVER_ERROR)
					}
				} else {
					event.Publish(notif.HTTP_SERVER_ERROR)
				}
			})
		}
		_, err = Send(HTTPSchema, HttpGET, address, pingPath, "")
		if err == nil {
			event.Publish(notif.HTTP_SERVER_STARTED)
		} else {
			event.Publish(notif.HTTP_SERVER_ERROR)
		}
	})
	return event
}

package connect

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"pulsyflux/msgbus"
	"pulsyflux/subscriptions"
	"pulsyflux/task"
	"pulsyflux/util"
	"strings"
)

type HttpRequest struct {
	Schema HttpSchema
	Method HttpMethod
	Host   string
	Port   int
	Path   string
	Data   string
}

type HttpResponse struct {
	Request       HttpRequest
	StatusCode    int
	StatusMessage string
	Data          string
}

func HttpRequestSubscriptions() {
	task.DoNow(msgbus.New(subscriptions.HTTP), func(httpCh *msgbus.Channel) any {

		httpMethod := task.DoNow(httpCh.New(subscriptions.HTTP_REQUEST), func(httpReqCh *msgbus.Channel) string {
			httpRequestMethodCh := httpReqCh.New(subscriptions.HTTP_REQUEST_METHOD)
			requestMethodMsg := httpRequestMethodCh.Subscribe()
			return requestMethodMsg.String()
		}, func(err error, httpReqCh *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_METHOD)
		})

		protocol := task.DoNow(httpCh.New(subscriptions.HTTP_REQUEST), func(httpReqCh *msgbus.Channel) string {
			requestProtocolCh := httpReqCh.New(subscriptions.REQUEST_PROTOCAL)
			protoMsg := requestProtocolCh.Subscribe()
			return protoMsg.String()
		}, func(err error, httpReqCh *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_REQUEST_PROTOCAL)
		})

		address := task.DoNow(httpCh.New(subscriptions.HTTP_REQUEST), func(httpReqCh *msgbus.Channel) *util.Address {
			httpReqAddressCh := httpReqCh.New(subscriptions.HTTP_REQUEST_ADDRESS)
			addressMsg := httpReqAddressCh.Subscribe()
			return util.NewAddress(addressMsg.String())
		}, func(err error, httpReqCh *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_ADDRESS)
		})

		requestURL := task.DoNow(httpCh.New(subscriptions.HTTP_REQUEST), func(httpReqCh *msgbus.Channel) *url.URL {
			httpReqPathCh := httpReqCh.New(subscriptions.HTTP_REQUEST_PATH)
			reqPathMsg := httpReqPathCh.Subscribe()
			addr := address.String()
			prefix := fmt.Sprintf("%s://", protocol)
			if !strings.HasPrefix(addr, prefix) {
				addr = prefix + addr
			}
			fullAddress := addr + reqPathMsg.String()
			url, err := url.ParseRequestURI(fullAddress)
			if err != nil {
				errorMsg := fmt.Sprintf("Could not parse url: %s, error: %v", fullAddress, err)
				err := errors.New(errorMsg)
				panic(err)
			}
			return url
		}, func(err error, httpReqCh *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_PATH)
		})

		reqBody := task.DoNow(httpCh.New(subscriptions.HTTP_REQUEST), func(httpReqCh *msgbus.Channel) io.Reader {
			httpRequestBodyCh := httpReqCh.New(subscriptions.HTTP_REQUEST_DATA)
			requestBodyMsg := httpRequestBodyCh.Subscribe()
			reqBody := util.ReaderFromString(requestBodyMsg.String())
			return reqBody
		}, func(err error, httpReqCh *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_DATA)
		})

		task.DoNow(httpCh.New(subscriptions.HTTP_REQUEST), func(httpReqCh *msgbus.Channel) any {
			httpReqCh.Subscribe()
			req, err := http.NewRequest(httpMethod, requestURL.String(), reqBody)
			if err != nil {
				panic(err)
			}
			res, err := http.DefaultClient.Do(req)
			req.Body.Close()
			req = nil
			if err != nil {
				panic(err)
			}
			resBody := util.StringFromReader(res.Body)
			resBodyMsg := msgbus.NewMessage(resBody)
			httpReqCh.Publish(resBodyMsg)
			return ""
		}, func(err error, httpReqCh *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST)
		})

		return nil
	}, func(err error, errorPub *msgbus.Channel) *msgbus.Channel {
		task.DoNow(errorPub, func(errorPub *msgbus.Channel) any {
			errorMsg := msgbus.NewMessage(err.Error())
			errorPub.Publish(errorMsg)
			return nil
		}, func(err error, errorPub *msgbus.Channel) *msgbus.Channel {
			fmt.Println("MessagePublishFail: could not publish the error message to the channel, logging the error here: ", err)
			return errorPub
		})
		return errorPub
	})
}

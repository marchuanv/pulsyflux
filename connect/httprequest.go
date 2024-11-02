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

	task.Do(func() (any, error) {

		httpCh := msgbus.New(subscriptions.HTTP)
		httpReqCh := httpCh.New(subscriptions.HTTP_REQUEST)

		httpMethod := task.Do(func() (string, error) {
			httpRequestMethodCh := httpReqCh.New(subscriptions.HTTP_REQUEST_METHOD)
			httpRequestMethod := httpRequestMethodCh.Subscribe()
			return httpRequestMethod.String(), nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_METHOD)
		})

		protocol := task.Do(func() (string, error) {
			requestProtocolCh := httpReqCh.New(subscriptions.REQUEST_PROTOCAL)
			httpScheme := requestProtocolCh.Subscribe()
			return httpScheme.String(), nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_REQUEST_PROTOCAL)
		})

		address := task.Do(func() (*util.Address, error) {
			receiveHttpAddress := httpReqCh.New(subscriptions.HTTP_REQUEST_ADDRESS)
			msg := receiveHttpAddress.Subscribe()
			addressStr := msg.String()
			address := util.NewAddress(addressStr)
			return address, nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_ADDRESS)
		})

		url := task.Do(func() (*url.URL, error) {
			httpReqPathCh := httpReqCh.New(subscriptions.HTTP_REQUEST_PATH)
			httpReqPathCh.Subscribe()
			url, err := url.ParseRequestURI(address.String())
			if err != nil {
				errorMsg := fmt.Sprintf("Could not parse url: %s, error: %v", address.String(), err)
				return nil, errors.New(errorMsg)
			}
			return url, nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_PATH)
		})

		reqBody := task.Do(func() (io.Reader, error) {
			httpRequestBodyCh := httpReqCh.New(subscriptions.HTTP_REQUEST_DATA)
			msg := httpRequestBodyCh.Subscribe()
			reqBody := util.ReaderFromString(msg.String())
			return reqBody, nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			invalidHttpAddress := httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_DATA)
			return invalidHttpAddress
		})

		addr := address.String()
		prefix := fmt.Sprintf("%s://", protocol)
		if !strings.HasPrefix(addr, prefix) {
			addr = prefix + addr
		}

		resBody := task.Do(func() (*http.Request, error) {
			httpReqCh.Subscribe()
			requestURL := fmt.Sprintf("%s://%s:%d%s", addr, address.Host, address.Port, url.Path)
			req, err := http.NewRequest(httpMethod, requestURL, reqBody)
			if err != nil {
				return nil, err
			}

			res, err := http.DefaultClient.Do(req)
			req.Body.Close()
			req = nil
			if err != nil {
				return response, err
			}
			resBody, err := util.StringFromReader(res.Body)
			if err != nil {
				return response, err
			}

			return resBody, nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			invalidHttpRequest := httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST)
			return invalidHttpRequest
		})

		resBody := task.Do(func() (*http.Request, error) {
			httpReqCh.Subscribe()
			requestURL := fmt.Sprintf("%s://%s:%d%s", addr, address.Host, address.Port, url.Path)
			req, err := http.NewRequest(httpMethod, requestURL, reqBody)
			if err != nil {
				return nil, err
			}

			res, err := http.DefaultClient.Do(req)
			req.Body.Close()
			req = nil
			if err != nil {
				return response, err
			}
			resBody, err := util.StringFromReader(res.Body)
			if err != nil {
				return response, err
			}

			return resBody, nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			invalidHttpRequest := httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST)
			return invalidHttpRequest
		})
		return nil, nil
	}, func(err error, errorPub *msgbus.Channel) *msgbus.Channel {
		task.Do(func() (any, error) {
			errorMsg := msgbus.NewMessage(err.Error())
			errorPub.Publish(errorMsg)
			return nil, nil
		}, func(err error, param any) any {
			fmt.Println("MessagePublishFail: could not publish the error message to the channel, logging the error here: ", err)
			return nil
		})
		return nil
	})
}

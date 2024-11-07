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
			requestMethodMsg := httpRequestMethodCh.Subscribe()
			return requestMethodMsg.String(), nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_METHOD)
		})

		protocol := task.Do(func() (string, error) {
			requestProtocolCh := httpReqCh.New(subscriptions.REQUEST_PROTOCAL)
			protoMsg := requestProtocolCh.Subscribe()
			return protoMsg.String(), nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_REQUEST_PROTOCAL)
		})

		address := task.Do(func() (*util.Address, error) {
			httpReqAddressCh := httpReqCh.New(subscriptions.HTTP_REQUEST_ADDRESS)
			addressMsg := httpReqAddressCh.Subscribe()
			address := util.NewAddress(addressMsg.String())
			return address, nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_ADDRESS)
		})

		requestURL := task.Do(func() (*url.URL, error) {
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
				return nil, errors.New(errorMsg)
			}
			return url, nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_PATH)
		})

		reqBody := task.Do(func() (io.Reader, error) {
			httpRequestBodyCh := httpReqCh.New(subscriptions.HTTP_REQUEST_DATA)
			requestBodyMsg := httpRequestBodyCh.Subscribe()
			reqBody := util.ReaderFromString(requestBodyMsg.String())
			return reqBody, nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_DATA)
		})

		task.Do(func() (any, error) {
			httpReqCh.Subscribe()
			req, err := http.NewRequest(httpMethod, requestURL.String(), reqBody)
			if err != nil {
				return "", err
			}
			res, err := http.DefaultClient.Do(req)
			req.Body.Close()
			req = nil
			if err != nil {
				return "", err
			}
			resBody := util.StringFromReader(res.Body)
			resBodyMsg := msgbus.NewMessage(resBody)
			httpReqCh.Publish(resBodyMsg)
			return "", nil
		}, func(err error, param *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST)
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

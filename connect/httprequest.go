package connect

import (
	"errors"
	"fmt"
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

	task.Do(func() (*msgbus.Channel, error) {

		httpCh := msgbus.New(subscriptions.HTTP)
		httpReqCh := httpCh.New(subscriptions.HTTP_REQUEST)

		httpMethod, errorPubCh := task.Do[string, *msgbus.Channel](func() (string, error) {
			httpRequestMethodCh := httpReqCh.New(subscriptions.HTTP_REQUEST_METHOD)
			httpRequestMethod := httpRequestMethodCh.Subscribe()
			return httpRequestMethod.String(), nil
		}, func(err error, params ...string) *msgbus.Channel {
			invalidHttpMethod := httpReqCh.New(subscriptions.INVALID_HTTP_METHOD)
			return invalidHttpMethod
		})
		if errorPubCh != nil {
			return errorPubCh, errors.New("invalid http method")
		}

		protocol, errorPubCh := task.Do[string, *msgbus.Channel](func() (string, error) {
			requestProtocolCh := httpReqCh.New(subscriptions.REQUEST_PROTOCAL)
			httpScheme := requestProtocolCh.Subscribe()
			return httpScheme.String(), nil
		}, func(err error, params ...string) *msgbus.Channel {
			invalidHttpMethod := httpReqCh.New(subscriptions.INVALID_HTTP_METHOD)
			return invalidHttpMethod
		})
		if errorPubCh != nil {
			return errorPubCh, errors.New("invalid protocal")
		}

		address, errorPubCh := task.Do[*util.Address, *msgbus.Channel](func() (*util.Address, error) {
			receiveHttpAddress := httpCh.New(subscriptions.RECEIVE_HTTP_ADDRESS)
			msg := receiveHttpAddress.Subscribe()
			addressStr := msg.String()
			address := util.NewAddress(addressStr)
			return address, nil
		}, func(err error, params ...*util.Address) *msgbus.Channel {
			invalidHttpAddress := httpCh.New(subscriptions.RECEIVE_HTTP_ADDRESS)
			return invalidHttpAddress
		})
		if errorPubCh != nil {
			return errorPubCh, errors.New("failed on server address")
		}

		path, errorPubCh := task.Do[string, *msgbus.Channel](func() (string, error) {
			receiveHttpRequestBodyCh := httpCh.New(subscriptions.HTTP_REQUEST_DATA)
			msg := receiveHttpRequestBodyCh.Subscribe()
			return msg.String(), nil
		}, func(err error, params ...string) *msgbus.Channel {
			invalidHttpAddress := httpCh.New(subscriptions.HTTP_REQUEST_DATA)
			return invalidHttpAddress
		})
		if errorPubCh != nil {
			return errorPubCh, errors.New("failed on server address")
		}

		path, errorPubCh := task.Do[string, *msgbus.Channel](func() (string, error) {
			receiveHttpRequestBodyCh := httpCh.New(subscriptions.HTTP_REQUEST_DATA)
			msg := receiveHttpRequestBodyCh.Subscribe()
			return msg.String(), nil
		}, func(err error, params ...string) *msgbus.Channel {
			invalidHttpAddress := httpCh.New(subscriptions.HTTP_REQUEST_DATA)
			return invalidHttpAddress
		})
		if errorPubCh != nil {
			return errorPubCh, errors.New("failed on server address")
		}

		return nil, nil
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

	httpRequestErrorResponse := httpReqCh.New(subscriptions.HTTP_REQUEST_ERROR_RESPONSE)
	httpRequestErrorResponse := httpReqCh.New(subscriptions.HTTP_REQUEST_ERROR_RESPONSE)

	request := HttpRequest{
		schema,
		method,
		"",
		-1,
		path,
		data,
	}
	response := HttpResponse{
		request,
		-1,
		"no response",
		"",
	}
	if util.IsEmptyString(request.Method.String()) {
		return response, errors.New("the method argument is an empty string")
	}
	if util.IsEmptyString(address) {
		return response, errors.New("the address argument is an empty string")
	}
	if util.IsEmptyString(request.Path) {
		return response, errors.New("the path argument is an empty string")
	}
	addr := address
	prefix := fmt.Sprintf("%s://", schema)
	if !strings.HasPrefix(addr, prefix) {
		addr = prefix + addr
	}
	url, err := url.ParseRequestURI(addr)
	if err != nil {
		errorMsg := fmt.Sprintf("Could not parse url: %s, error: %v", addr, err)
		return response, errors.New(errorMsg)
	}
	host, port, err := util.NewAddress(url.Host)
	if err != nil {
		errorMsg := fmt.Sprintf("error making http request could not determine the host and port from address: %s\n", err)
		return response, errors.New(errorMsg)
	}
	request.Host = host
	request.Port = port
	reqBody, err := util.ReaderFromString(request.Data)
	if err != nil {
		errorMsg := fmt.Sprintf("error writing request data: %s\n", err)
		return response, errors.New(errorMsg)
	}
	requestURL := fmt.Sprintf("http://%s:%d%s", request.Host, request.Port, request.Path)
	req, err := http.NewRequest(method.String(), requestURL, reqBody)
	reqBody = nil
	if err != nil {
		errorMsg := fmt.Sprintf("could not create a new http request: %s\n", err)
		return response, errors.New(errorMsg)
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
	response.Data = resBody
	response.StatusCode = res.StatusCode
	response.StatusMessage = res.Status
	return response, nil
}
func Send(schema HttpSchema, method HttpMethod, address string, path string, data string) (HttpResponse, error) {
}

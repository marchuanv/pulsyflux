package connect

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

func Send(schema HttpSchema, method HttpMethod, address string, path string, data string) (HttpResponse, error) {
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
	host, port, err := util.GetHostAndPortFromAddress(url.Host)
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

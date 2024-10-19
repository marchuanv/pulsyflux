package connect

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"pulsyflux/util"
	"strings"
)

type Schema int

const (
	HTTPSchema Schema = iota
	HTTPSSchema
)

func (sch Schema) String() string {
	switch sch {
	case HTTPSchema:
		return "http"
	case HTTPSSchema:
		return "https"
	default:
		return fmt.Sprintf("%d", int(sch))
	}
}

type HttpRequest struct {
	schema Schema
	method string
	host   string
	port   int
	path   string
	data   string
}

type HttpResponse struct {
	request       HttpRequest
	statusCode    int
	statusMessage string
	data          string
}

func Send(schema Schema, method string, address string, path string, data string) (HttpResponse, error) {
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
	if util.IsEmptyString(request.method) {
		return response, errors.New("the method argument is an empty string")
	}
	if util.IsEmptyString(address) {
		return response, errors.New("the address argument is an empty string")
	}
	if util.IsEmptyString(request.path) {
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
	request.host = host
	request.port = port
	reqBody, err := util.ReaderFromString(request.data)
	if err != nil {
		errorMsg := fmt.Sprintf("error writing request data: %s\n", err)
		return response, errors.New(errorMsg)
	}
	requestURL := fmt.Sprintf("http://%s:%d%s", request.host, request.port, request.path)
	req, err := http.NewRequest(method, requestURL, reqBody)
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
	response.data = resBody
	response.statusCode = res.StatusCode
	response.statusMessage = res.Status
	return response, nil
}

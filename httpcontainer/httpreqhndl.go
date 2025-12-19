package httpcontainer

import (
	"net/http"
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
	"pulsyflux/util"
)

type httpRequestHandler struct {
	responses *sliceext.List[contracts.HttpResponse]
}

func (rh *httpRequestHandler) Init(container contracts.Container1[registeredResponses, *registeredResponses]) {
	rh.responses = container.Get().list
}

func (rh *httpRequestHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if rh.responses.Len() == 0 {
		http.Error(response, "no http responses configured", http.StatusInternalServerError)
		return
	}
	reqBody := util.StringFromReader(request.Body)
	var reason string
	var statusCode int
	var resBody string
	for _, httpRes := range rh.responses.All() {
		reason, statusCode, resBody = httpRes.Handle(request.Header, reqBody)
		if statusCode == *httpRes.GetSuccessStatusCode() {
			response.WriteHeader(statusCode)
			response.Write([]byte(resBody))
			return
		}
	}
	http.Error(response, reason, statusCode)
}

func NewHttpRequestContainer() contracts.Container2[httpRequestHandler, registeredResponses, *registeredResponses, *httpRequestHandler] {
	c := NewRegResContainer()
	return containers.NewContainer2[httpRequestHandler, registeredResponses, *registeredResponses, *httpRequestHandler](c)
}

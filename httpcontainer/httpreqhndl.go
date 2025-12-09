package httpcontainer

import (
	"net/http"
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
	"pulsyflux/util"
)

type httpRequestHandler struct {
	httpResponseIds *sliceext.List[contracts.TypeId[contracts.HttpResponse]]
}

func (rh *httpRequestHandler) SetHttpResponseIds(httpResponseIds *sliceext.List[contracts.TypeId[contracts.HttpResponse]]) {
	rh.httpResponseIds = httpResponseIds
}

func (rh *httpRequestHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if rh.httpResponseIds.Len() == 0 {
		http.Error(response, "no http responses configured", http.StatusInternalServerError)
		return
	}
	reqBody := util.StringFromReader(request.Body)
	var reason string
	var statusCode int
	var resBody string
	for _, httpResId := range rh.httpResponseIds.All() {
		httpRes := containers.Get[contracts.HttpResponse](httpResId)
		reason, statusCode, resBody = httpRes.Handle(request.Header, reqBody)
		if statusCode == *httpRes.GetSuccessStatusCode() {
			response.WriteHeader(statusCode)
			response.Write([]byte(resBody))
			return
		}
	}
	http.Error(response, reason, statusCode)
}

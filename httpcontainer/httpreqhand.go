package httpcontainer

import (
	"net/http"
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
	"pulsyflux/util"
)

type httpReqHandler struct {
	responses *sliceext.List[*httpResHandler]
}

// Init resolves the registeredResponses container
func (rh *httpReqHandler) Init(container contracts.Container1[registeredResponses, *registeredResponses]) {
	rh.responses = container.Get().list
}

// ServeHTTP now propagates the request context to each response handler
func (rh *httpReqHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rh.responses.Len() == 0 {
		http.Error(w, "no http responses configured", http.StatusInternalServerError)
		return
	}

	// Read request body safely
	reqBody := util.StringFromReader(r.Body)

	var reason string
	var statusCode int
	var resBody string

	// Iterate over all registered response handlers
	for _, res := range rh.responses.All() {
		// Pass the HTTP request context to the handler
		reason, statusCode, resBody = res.handle(r.Context(), r.Header, reqBody)

		// If handler returns success, write the response and exit
		if statusCode == res.successStatusCode {
			w.WriteHeader(statusCode)
			w.Write([]byte(resBody))
			return
		}
	}

	// If none matched, return last reason/status
	http.Error(w, reason, statusCode)
}

// Factory function for DI container
func NewHttpRequestContainer() contracts.Container2[httpReqHandler, registeredResponses, *registeredResponses, *httpReqHandler] {
	c := NewRegResContainer()
	return containers.NewContainer2[httpReqHandler, registeredResponses, *registeredResponses, *httpReqHandler](c)
}

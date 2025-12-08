package httpcontainers

import (
	"net/http"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
)

type httpRequestHandler struct {
	handlers *sliceext.List[contracts.HttpRequest]
}

func (conn *httpRequestHandler) GetHandlers() *sliceext.List[contracts.HttpRequest] {
	return conn.handlers
}

func (conn *httpRequestHandler) SetHandlers(handlers *sliceext.List[contracts.HttpRequest]) {
	conn.handlers = handlers
}

func (conn *httpRequestHandler) Receive(recv func(envelope contracts.Envelope)) {
	conn.handlers.Add(func(response http.ResponseWriter, request *http.Request) {
		// msgArg := Arg{"ReceivedHttpMessage", util.StringFromReader(request.Body)}
		// nvlp := RegisterEnvlpFactory().get(msgArg)
		response.WriteHeader(http.StatusOK)
	})
}

func (conn *httpRequestHandler) Send(envelope contracts.Envelope) {
}

func (conn *httpRequestHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	for _, handler := range conn.GetHandlers().All() {
		handler(response, request)
	}
}

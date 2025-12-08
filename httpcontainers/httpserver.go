package containers

import (
	"net/http"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
	"time"
)

const (
	httpConnFactory factory[contracts.Connection] = "02c88068-3a66-4856-b2bf-e2dce244761b"
)

type HttpRequest func(response http.ResponseWriter, request *http.Request)

type httpConnection struct {
	handlers *sliceext.Stack[HttpRequest]
	server   *http.Server
}

func RegisterConnFactory() factory[contracts.Connection] {
	httpConnFactory.register(func(args ...Arg) contracts.Connection {
		isUri, uri := argValue[contracts.URI](&args[0])
		if isUri {
			conn := &httpConnection{}
			conn.handlers = sliceext.NewStack[HttpRequest]()
			conn.server = &http.Server{
				Addr:           uri.String(),
				ReadTimeout:    10 * time.Second,
				WriteTimeout:   10 * time.Second,
				MaxHeaderBytes: 1 << 20,
				Handler:        conn,
			}
			return conn
		}
		return nil
	})
	return httpConnFactory
}

func (conn *httpConnection) State() contracts.ConnectionState {
	return nil
}
func (conn *httpConnection) Open() {

}
func (conn *httpConnection) Close() {

}
func (conn *httpConnection) Receive(recv func(envelope contracts.Envelope)) {
	conn.handlers.Push(func(response http.ResponseWriter, request *http.Request) {
		// msgArg := Arg{"ReceivedHttpMessage", util.StringFromReader(request.Body)}
		// nvlp := RegisterEnvlpFactory().get(msgArg)
		response.WriteHeader(http.StatusOK)
	})
}
func (conn *httpConnection) Send(envelope contracts.Envelope) {
}
func (conn *httpConnection) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	clonedHandlers := conn.handlers.Clone()
	for clonedHandlers.Len() > 0 {
		handler := clonedHandlers.Pop()
		handler(response, request)
	}
}

package factory

import (
	"net/http"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
	"pulsyflux/util"
	"time"
)

const (
	HttpConnFactory Factory[contracts.Connection] = "02c88068-3a66-4856-b2bf-e2dce244761b"
)

type HttpRequest func(response http.ResponseWriter, request *http.Request)
type ReceiveHttpRequest func(recv HttpRequest)

type envelope struct {
	value any
}

func (envlp *envelope) Content() any {

}

type httpConnection struct {
	handlers *sliceext.Stack[HttpRequest]
	server   *http.Server
}

func (conn *httpConnection) Open() {

}
func (conn *httpConnection) Close() {

}
func (conn *httpConnection) Receive(recv func(envelope contracts.Envelope)) {
	conn.handlers.Push(func(response http.ResponseWriter, request *http.Request) {
		envlp := envelope{util.StringFromReader(request.Body)}
		// _contentType := request.Header.Get("content-type")
		// _auth := request.Header.Get("Authorization")
		response.WriteHeader(http.StatusOK)
		recv(&envlp)
	})

}
func (conn *httpConnection) Send(envelope contracts.Envelope) {

}

func (handler *httpConnection) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	clonedHandlers := handler.handlers.Clone()
	for clonedHandlers.Len() > 0 {
		handler := clonedHandlers.Pop()
		handler(response, request)
	}
}

func init() {
	HttpConnFactory.Ctor(func(args []*Arg) contracts.Connection {
		isUri, uri := ArgValue[contracts.URI](args[0])
		if isUri {
			conn := &httpConnection{}
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
}

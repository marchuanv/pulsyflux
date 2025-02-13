package events

import (
	"net"
	"net/http"
	"net/url"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
	"strconv"
	"time"
)

const (
	CreateURL             Event[string, *url.URL]                           = "ff620615-bdd1-4f41-b570-a9d8c83d8c79"
	CreateURI             Event[contracts.URI, contracts.URI]               = "ec37e9cc-7931-47a2-a948-d20906e387ab"
	ConvertToURLEvent     Event[contracts.URI, *url.URL]                    = "725fce3e-4215-4a8b-b7cf-2af95c9be727"
	ConvertToURIEvent     Event[*url.URL, contracts.URI]                    = "32ca0258-3119-4635-8caa-e9a9b7ce2ba7"
	CreateConnectionEvent Event[contracts.URI, contracts.Connection]        = "9ccc55de-58c1-4e1a-9f00-0c335695db27"
	OpenConnectionEvent   Event[contracts.Connection, contracts.Connection] = "6b7cb263-af03-4440-b7a8-f0904cf1eecf"
	CloseConnectionEvent  Event[contracts.Connection, contracts.Connection] = "ca449035-c3c4-4727-982a-3cc81ca979ba"
	ReceiveEnvelopeEvent  Event[contracts.Connection, contracts.Envelope]   = "70693e8e-07ec-4106-96ad-367e2e834742"
	SendEnvelopeEvent     Event[contracts.Connection, contracts.Envelope]   = "b3b81216-1a94-4adf-9754-010c78202ab2"
)

func FactorySubscriptions() {
	ConvertToURLEvent.New(func(_uri contracts.URI) *url.URL {
		_url, err := url.Parse(_uri.String())
		if err != nil {
			panic(err)
		}
		return _url
	})
	ConvertToURIEvent.New(func(url *url.URL) contracts.URI {
		conv := uri{
			url.Scheme,
			url.Host,
			url.Path,
			0,
		}
		_port, err := strconv.Atoi(url.Port())
		if err != nil {
			panic(err)
		}
		conv.port = _port
		return URI(conv)
	})
	CreateHttpServerEvent.New(func(u URI) *http.Server {
		return &http.Server{
			Addr:           u.String(),
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
			Handler: &HttpRequestHandler{
				sliceext.NewStack[HttpRequest](),
			},
		}
	})
	ReceiveHttpServerRequestHandlerEvent.New(func(server *http.Server) *HttpRequestHandler {
		return server.Handler.(*HttpRequestHandler)
	})
	ReceiveHttpRequestEvent.New(func(handler *HttpRequestHandler) ReceiveHttpRequest {
		return func(recv HttpRequest) {
			handler.handlers.Push(recv)
		}
	})
	StartHttpServerEvent.New(func(httpServer *http.Server) net.Listener {
		_listener, err := net.Listen("tcp", httpServer.Addr)
		if err != nil {
			panic(err)
		}
		err = httpServer.Serve(_listener)
		if err != nil {
			panic(err)
		}
		return _listener
	})
}

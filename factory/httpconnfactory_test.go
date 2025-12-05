package factory

import (
	"net/url"
	"pulsyflux/contracts"
	"testing"
	"time"
)

func TestHttpConnFactory(test *testing.T) {
	urlStrArg := &Arg{"hostURLStr", "http://localhost:3000"}
	URLFactory().get(func(url *url.URL) {
		urlArg := &Arg{"hostURL", url}
		URIFactory().get(func(uri contracts.URI) {
			uriArg := &Arg{"hostURI", uri}
			ConnFactory().get(func(conn contracts.Connection) {
				conn.Receive(func(envelope contracts.Envelope) {
					test.Log(envelope.Content())
				})
				msgArg := &Arg{"message", "Hello World"}
				EnvlpFactory().get(func(obj contracts.Envelope) {
					conn.Send(obj)
				}, msgArg)
			}, uriArg)
		}, urlArg)
	}, urlStrArg)
	time.Sleep(1000 * time.Millisecond)
}

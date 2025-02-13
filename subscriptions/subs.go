package subscriptions

import (
	"net"
	"net/url"
	"pulsyflux/channel"
)

type URI interface {
	Host() string
	Port() string
	Path() string
	String() string
}

type HostURI interface {
	Host() string
	Port() string
	Path() string
	String() string
}

type HttpSchema string

const (
	HTTP  HttpSchema = "aab627c3-4fe3-4ce7-b893-a034cfe8074e"
	HTTPS HttpSchema = "ce1fd105-38bb-4a37-8af0-5ae94431eafe"
)

func (schema HttpSchema) String() string {
	switch schema {
	case "aab627c3-4fe3-4ce7-b893-a034cfe8074e":
		return "http://"
	case "ce1fd105-38bb-4a37-8af0-5ae94431eafe":
		return "https://"
	default:
		return ""
	}
}

type HttpMethod string

const (
	HttpGET  HttpMethod = "77ea1c80-551d-4aee-897d-86f20747e436"
	HttpPOST HttpMethod = "37b7b7e3-3a36-47a4-a0a9-9b1e7d9cd8fd"
)

func (method HttpMethod) String() string {
	switch method {
	case "77ea1c80-551d-4aee-897d-86f20747e436":
		return "GET"
	case "37b7b7e3-3a36-47a4-a0a9-9b1e7d9cd8fd":
		return "POST"
	default:
		return ""
	}
}

var URIChannel = channel.NewChnlId("5072497a-a035-4a6a-8cd9-a76d4b1e9c0e")
var RequestSchemaChannel = channel.NewChnlId("b853d7f6-a9d9-4a39-8b61-9ccd14033f06")
var HostURIChannel = channel.NewChnlId("e0839492-dc24-4795-bd24-4431cc0498c7")
var HttpListenerChannel = channel.NewChnlId("30f9d8ab-22d6-4029-a034-553b8136c1c1")
var HttpRequestChannel = channel.NewChnlId("a534fcc1-c59e-4752-9dc5-8cda948b8bf3")
var RequestMethodChannel = channel.NewChnlId("185d2aa7-94a7-408f-87c0-710db30e814c")

func URISubscription(receive func(uri URI)) {
	channel.Subscribe(URIChannel, func(uri URI) {
		receive(uri)
	})
}

func RequestSchemaSubscription(receive func(httpSchema string)) {
	channel.Subscribe(RequestSchemaChannel, func(httpSchema HttpSchema) {
		receive(httpSchema.String())
	})
}

func RequestMethodSubscription(receive func(httpMethod string)) {
	channel.Subscribe(RequestMethodChannel, func(httpReqMethod HttpMethod) {
		receive(httpReqMethod.String())
	})
}

func HttpRequestSubscription(receive func(url *url.URL)) {
	RequestSchemaSubscription(func(httpSchema string) {
		channel.Subscribe(HttpRequestChannel, func(url *url.URL) {
			receive(url)
		})
	})
}

func HostURISubscription(receive func(uri HostURI)) {
	channel.Subscribe(HostURIChannel, func(uri HostURI) {
		receive(uri)
	})
}

func HttpListenerSubscription(receive func(listener net.Listener, hostURI HostURI)) {
	HostURISubscription(func(hostURI HostURI) {
		channel.Subscribe(HttpListenerChannel, func(listener net.Listener) {
			receive(listener, hostURI)
		})
	})
}

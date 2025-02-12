package subscriptions

import (
	"errors"
	"pulsyflux/channel"
)

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

func SubscribeToHttpRequestMethod(chnlId channel.ChnlId, receive func(httpMethod string)) {
	channel.Subscribe(chnlId, func(httpReqMethod HttpMethod) {
		receive(httpReqMethod.String())
	})
}

func PublishHttpRequestMethod(chnlId channel.ChnlId, httpReqMethod HttpMethod) {
	if httpReqMethod.String() == "" {
		channel.Publish(chnlId, errors.New("invalid http request method"))
	} else {
		channel.Publish(chnlId, httpReqMethod)
	}
}

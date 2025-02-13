package subscriptions

import (
	"errors"
	"pulsyflux/channel"
)

func PublishHttpRequestMethod(chnlId channel.ChnlId, httpReqMethod HttpMethod) {
	if httpReqMethod.String() == "" {
		channel.Publish(chnlId, errors.New("invalid http request method"))
	} else {
		channel.Publish(chnlId, httpReqMethod)
	}
}

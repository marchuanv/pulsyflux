package subscriptions

import (
	"errors"
	"fmt"
	"net/url"
	"pulsyflux/channel"
)

func PublishHttpRequest(chnlId channel.ChnlId) {
	SubscribeToHttpSchema(chnlId, func(httpSchema string) {
		SubscribeToHttpRequestMethod(chnlId, func(httpMethod string) {
			SubscribeToAddress(chnlId, func(addr *Address) {
				address := addr.String()
				fullAddress := httpSchema + address + reqPathMsg.String()
				url, err := url.ParseRequestURI(fullAddress)
				if err != nil {
					errorMsg := fmt.Sprintf("Could not parse url: %s, error: %v", fullAddress, err)
					panic(errors.New(errorMsg))
				}
				channel.Publish(chnlId, url)
			})
		})
	})
}

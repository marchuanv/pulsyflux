package subscriptions

import (
	"errors"
	"pulsyflux/channel"
)

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

func SubscribeToHttpSchema(chnlId channel.ChnlId, receive func(httpSchema string)) {
	channel.Subscribe(chnlId, func(httpSchema HttpSchema) {
		receive(httpSchema.String())
	})
}

func PublishHttpSchema(chnlId channel.ChnlId, httpSchema HttpSchema) {
	if httpSchema.String() == "" {
		channel.Publish(chnlId, errors.New("invalid http schema"))
	} else {
		channel.Publish(chnlId, httpSchema)
	}
}

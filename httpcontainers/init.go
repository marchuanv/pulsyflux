package httpcontainers

import (
	"net/url"
	"pulsyflux/containers"
	"pulsyflux/contracts"
)

type envelope struct {
	url *url.URL
	msg func() any
}

const (
	envelopeFactory contracts.TypeId[contracts.Envelope] = "f9b8f074-5883-4888-94a2-91a9fad2c6ac"
)

func init() {
	containers.RegisterType(envelopeFactory)
}

func RegisterEnvlpFactory() contracts.Container2[contracts.Envelope] {
	envelopeFactory.register(func(args ...Arg) contracts.Envelope {
		_, url := argValue[*url.URL](&args[0])
		_, msg := argValue[any](&args[1])
		return &envelope{url, func() any {
			return msg
		}}
	})
	return envelopeFactory
}

func (envlp *envelope) Content() any {
	return envlp.msg()
}
func (envlp *envelope) Address() *url.URL {
	return envlp.url
}

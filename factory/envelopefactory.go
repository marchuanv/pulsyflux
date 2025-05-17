package factory

import (
	"pulsyflux/contracts"
)

const (
	envelopeFactory factory[contracts.Envelope] = "f9b8f074-5883-4888-94a2-91a9fad2c6ac"
)

type envelope struct {
	msg func() any
}

func EnvlpFactory() factory[contracts.Envelope] {
	envelopeFactory.ctor(func(args ...*Arg) contracts.Envelope {
		_, msg := argValue[any](args[0])
		return &envelope{func() any {
			return msg
		}}
	})
	return envelopeFactory
}

func (envlp *envelope) Content() any {
	return envlp.msg()
}

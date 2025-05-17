package factory

import (
	"pulsyflux/channel"

	"github.com/google/uuid"
)

var factoryChnlCtor = channel.NewChnlId("dd730e01-6d8b-409b-8d7b-e2ec873b8adf")
var factoryChnlGet = channel.NewChnlId("0ca0e184-2f34-4fec-a297-7825ec2b4761")

type factory[T interface{}] string

type factoryCtor[T interface{}] struct {
	id   uuid.UUID
	args func() []*Arg
}
type factoryObj[T interface{}] struct {
	id  uuid.UUID
	get func() T
}

type Arg struct {
	name  string
	value any
}

func argValue[T any](arg *Arg) (canConvert bool, content T) {
	defer (func() {
		err := recover()
		if err != nil {
			canConvert = false
		}
	})()
	content = arg.value.(T)
	canConvert = true
	return canConvert, content
}

func (f factory[T]) ctor(ctor func(args ...*Arg) T) {
	if !channel.HasChnl(factoryChnlCtor) {
		channel.OpenChnl(factoryChnlCtor)
	}
	if !channel.HasChnl(factoryChnlGet) {
		channel.OpenChnl(factoryChnlGet)
	}
	channel.Subscribe(factoryChnlCtor, func(fact factoryCtor[T]) {
		if fact.id.String() == string(f) {
			channel.Publish(factoryChnlGet, factoryObj[T]{
				fact.id,
				func() T {
					return ctor(fact.args()...)
				},
			})
		}
	})
}

func (f factory[T]) get(recv func(obj T), args ...*Arg) {
	if !channel.HasChnl(factoryChnlCtor) {
		channel.OpenChnl(factoryChnlCtor)
	}
	if !channel.HasChnl(factoryChnlGet) {
		channel.OpenChnl(factoryChnlGet)
	}
	channel.Subscribe(factoryChnlGet, func(fact factoryObj[T]) {
		if fact.id.String() == string(f) {
			recv(fact.get())
		}
	})
	channel.Publish(factoryChnlCtor, factoryCtor[T]{
		uuid.MustParse(string(f)),
		func() []*Arg { return args },
	})
}

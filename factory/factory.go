package factory

import (
	"pulsyflux/channel"

	"github.com/google/uuid"
)

var factoryChnl = channel.NewChnlId("dd730e01-6d8b-409b-8d7b-e2ec873b8adf")

type Factory[T interface{}] string

type factoryCtor[T interface{}] struct {
	id   uuid.UUID
	args func() []*Arg
}
type factoryObj[T interface{}] struct {
	id  uuid.UUID
	get func() T
}

type Arg struct {
	value any
}

func ArgValue[T any](arg *Arg) (canConvert bool, content T) {
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

func (f Factory[T]) Ctor(ctor func(args []*Arg) T) {
	channel.Subscribe(factoryChnl, func(fact factoryCtor[T]) {
		if fact.id.String() == string(f) {
			channel.Publish(factoryChnl, factoryObj[T]{
				fact.id,
				func() T {
					return ctor(fact.args())
				},
			})
		}
	})
}

func (f Factory[T]) Get(recv func(obj T), args []*Arg) {
	channel.Subscribe(factoryChnl, func(fact factoryObj[T]) {
		if fact.id.String() == string(f) {
			recv(fact.get())
		}
	})
	channel.Publish(factoryChnl, factoryCtor[T]{
		uuid.MustParse(string(f)),
		func() []*Arg { return args },
	})
}

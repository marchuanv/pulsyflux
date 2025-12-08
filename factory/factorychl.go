package factory

import (
	"pulsyflux/channel"

	"github.com/google/uuid"
)

type factory[T interface{}] string

func (f factory[T]) register(ctor func(args ...Arg) T) {
	factoryChlId := string(f)
	factoryArgsChl := channel.GetChl[[]Arg](factoryChlId)
	if !factoryArgsChl.IsOpen() {
		factoryArgsChl.Open()
	}
	factoryArgsChlSub := channel.GetChlSub[[]Arg](uuid.NewString())
	factoryArgsChlSub.Subscribe(factoryArgsChl, func(args []Arg) {
		factoryGetObjChl := channel.GetChl[T](factoryChlId)
		if !factoryGetObjChl.IsOpen() {
			factoryGetObjChl.Open()
		}
		instance := ctor(args...)
		factoryGetObjChl.Publish(instance)
		factoryArgsChlSub.Unsubscribe(factoryArgsChl)
	})
}

func (f factory[T]) get(args ...Arg) T {
	factoryChlId := string(f)
	factoryArgsChl := channel.GetChl[[]Arg](factoryChlId)
	if !factoryArgsChl.IsOpen() {
		factoryArgsChl.Open()
	}
	factoryArgsChl.Publish(args)
	factoryGetObjChl := channel.GetChl[T](factoryChlId)
	if !factoryGetObjChl.IsOpen() {
		factoryGetObjChl.Open()
	}
	factoryGetObjSubChl := channel.GetChlSub[T](uuid.NewString())
	results := make(chan T)
	factoryGetObjSubChl.Subscribe(factoryGetObjChl, func(msg T) {
		results <- msg
	})
	return <-results
}

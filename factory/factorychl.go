package factory

import (
	"pulsyflux/channel"

	"github.com/google/uuid"
)

type Arg struct {
	name  string
	value any
}

type factory[T interface{}] string

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

func (f factory[T]) get(rcvObj func(obj T), args ...Arg) {
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
	factoryGetObjSubChl.Subscribe(factoryGetObjChl, rcvObj)
}

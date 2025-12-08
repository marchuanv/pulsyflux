package containers

import (
	"pulsyflux/contracts"
	"testing"
)

type Wheel interface {
	GetFront() bool
	SetFront(v bool)
}

type wheel struct {
	isFront bool
}

func (whl *wheel) GetFront() bool {
	return whl.isFront
}
func (whl *wheel) SetFront(v bool) {
	whl.isFront = v
}

type Car interface {
	GetWheel01() Wheel
	GetWheel02() Wheel
	GetWheel03() Wheel
	GetWheel04() Wheel
	SetWheel01(whl Wheel)
	SetWheel02(whl Wheel)
	SetWheel03(whl Wheel)
	SetWheel04(whl Wheel)
}

type car struct {
	wheel01 *wheel
	wheel02 *wheel
	wheel03 *wheel
	wheel04 *wheel
}

func (car *car) GetWheel01() Wheel {
	return car.wheel01
}
func (car *car) GetWheel02() Wheel {
	return car.wheel02
}
func (car *car) GetWheel03() Wheel {
	return car.wheel03
}
func (car *car) GetWheel04() Wheel {
	return car.wheel04
}

func (car *car) SetWheel01(whl Wheel) {
	car.wheel01 = whl.(*wheel)
}
func (car *car) SetWheel02(whl Wheel) {
	car.wheel02 = whl.(*wheel)
}
func (car *car) SetWheel03(whl Wheel) {
	car.wheel03 = whl.(*wheel)
}
func (car *car) SetWheel04(whl Wheel) {
	car.wheel04 = whl.(*wheel)
}

const (
	carType    contracts.TypeId[car]   = "f9b8f074-5883-4888-94a2-91a9fad2c6ac"
	wheel1Type contracts.TypeId[wheel] = "d1b01146-7282-4e58-ac45-a684bf1ebcfa"
	wheel2Type contracts.TypeId[wheel] = "0cc4d2b9-0e24-477f-98bb-102fbfdeb137"
	wheel3Type contracts.TypeId[wheel] = "c571f0b2-3b2d-4635-a0bf-dedefb02c3b2"
	wheel4Type contracts.TypeId[wheel] = "e0b206d6-f3e5-46da-b5a2-2534291730d6"
)

func TestContainer(test *testing.T) {
	RegisterType(carType)
	RegisterTypeDependency(carType, wheel1Type, "wheel01", nil)
	RegisterTypeDependency(carType, wheel2Type, "wheel02", &wheel{true})
	RegisterTypeDependency(carType, wheel3Type, "wheel03", nil)
	RegisterTypeDependency(carType, wheel4Type, "wheel04", nil)

	instance := Get[Car](carType)

	instance.GetWheel01().SetFront(true)
	instance.GetWheel03().SetFront(false)
	instance.GetWheel04().SetFront(false)

	ref := Get[Car](carType)

	if !ref.GetWheel01().GetFront() {
		test.Log("expected Wheel01.isFront to be true")
		test.Fail()
	}
	if !ref.GetWheel02().GetFront() {
		test.Log("expected Wheel02.isFront to be true")
		test.Fail()
	}

	wheel01 := Get[Wheel](wheel1Type)
	if !wheel01.GetFront() {
		test.Log("expected Wheel01.isFront to be true")
		test.Fail()
	}

	wheel02 := Get[Wheel](wheel2Type)
	if !wheel02.GetFront() {
		test.Log("expected Wheel02.isFront to be true")
		test.Fail()
	}

	wheel03 := Get[Wheel](wheel3Type)
	if wheel03.GetFront() {
		test.Log("expected Wheel03.isFront to be false")
		test.Fail()
	}

	wheel04 := Get[Wheel](wheel4Type)
	if wheel04.GetFront() {
		test.Log("expected Wheel04.isFront to be false")
		test.Fail()
	}
}

package containers

import (
	"pulsyflux/contracts"
	"testing"
)

type wheel struct {
	isFront bool
}
type car struct {
	Wheel01 *wheel
	Wheel02 *wheel
	Wheel03 *wheel
	Wheel04 *wheel
}

const (
	carType    contracts.TypeId[car]   = "f9b8f074-5883-4888-94a2-91a9fad2c6ac"
	wheel1Type contracts.TypeId[wheel] = "d1b01146-7282-4e58-ac45-a684bf1ebcfa"
	wheel2Type contracts.TypeId[wheel] = "0cc4d2b9-0e24-477f-98bb-102fbfdeb137"
	wheel3Type contracts.TypeId[wheel] = "c571f0b2-3b2d-4635-a0bf-dedefb02c3b2"
	wheel4Type contracts.TypeId[wheel] = "e0b206d6-f3e5-46da-b5a2-2534291730d6"
)

func TestContainer(test *testing.T) {
	Type(carType)
	TypeArg(carType, wheel1Type, "Wheel01", &wheel{})
	TypeArg(carType, wheel2Type, "Wheel02", &wheel{true})
	TypeArg(carType, wheel3Type, "Wheel03", &wheel{})
	TypeArg(carType, wheel4Type, "Wheel04", &wheel{})

	instance := Get(carType)

	instance.Wheel01.isFront = true
	instance.Wheel03.isFront = false
	instance.Wheel04.isFront = false

	ref := Get(carType)

	if !ref.Wheel01.isFront {
		test.Log("expected Wheel01.isFront to be true")
		test.Fail()
	}
	if !ref.Wheel02.isFront {
		test.Log("expected Wheel02.isFront to be true")
		test.Fail()
	}

	wheel01 := Get(wheel1Type)
	if !wheel01.isFront {
		test.Log("expected Wheel01.isFront to be true")
		test.Fail()
	}

	wheel02 := Get(wheel2Type)
	if !wheel02.isFront {
		test.Log("expected Wheel02.isFront to be true")
		test.Fail()
	}

	wheel03 := Get(wheel3Type)
	if wheel03.isFront {
		test.Log("expected Wheel03.isFront to be false")
		test.Fail()
	}

	wheel04 := Get(wheel4Type)
	if wheel04.isFront {
		test.Log("expected Wheel04.isFront to be false")
		test.Fail()
	}
}

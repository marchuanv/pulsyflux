package containers

import (
	"pulsyflux/contracts"
)

func Type[T comparable](typeId contracts.TypeId[T]) {
	addType(typeId, nil)
}

func TypeArg[T comparable, ArgT comparable](typeId contracts.TypeId[T], argTypeId contracts.TypeId[ArgT], argName string, value *ArgT) {
	addArgType(typeId, argTypeId, argName, value)
}

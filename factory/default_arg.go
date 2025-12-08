package factory

type arg[T any] struct {
	value T
}

func (a arg[T]) Get() T {
	return a.value
}

func (a arg[T]) Set(value T) {
	a.value = value
}

func NewArg[T any]() Arg[T] {
	return &arg[T]{}
}

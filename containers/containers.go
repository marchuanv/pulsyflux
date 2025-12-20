package containers

import "pulsyflux/contracts"

// 1. How your Container1-4 system works

// Each container (c1, c2, c3, c4) stores a pointer to a single instance (instance *T).

// When you call Get():

// If instance is nil, the container:

// Allocates a zero-value

// Casts it to the pointer interface PT(&val)

// Calls Init(...) automatically

// Stores it in instance

// After this, Get() always returns a fully initialized object.

// Implication: You cannot get an uninitialized handler through the container. That defensive nil check in Handle() is now largely redundant if you always use the container to resolve your handler.

// container1 implementation
type c1[T any, PT interface {
	*T
	Init()
}] struct {
	instance *T
}

func NewContainer1[T any, PT interface {
	*T
	Init()
}]() contracts.Container1[T, PT] {
	return &c1[T, PT]{}
}

func (c *c1[T, PT]) Get() T {
	if c.instance == nil {
		var val T
		PT(&val).Init()
		c.instance = &val
	}
	return *c.instance
}

// container2 implementation
type c2[T any, A1 any, PA1 interface {
	*A1
	Init()
}, PT interface {
	*T
	Init(contracts.Container1[A1, PA1])
}] struct {
	instance *T
	dep1     contracts.Container1[A1, PA1]
}

func NewContainer2[T any, A1 any, PA1 interface {
	*A1
	Init()
}, PT interface {
	*T
	Init(contracts.Container1[A1, PA1])
}](arg1 contracts.Container1[A1, PA1]) contracts.Container2[T, A1, PA1, PT] {
	return &c2[T, A1, PA1, PT]{dep1: arg1}
}

func (c *c2[T, A1, PA1, PT]) Get() T {
	if c.instance == nil {
		var val T
		PT(&val).Init(c.dep1)
		c.instance = &val
	}
	return *c.instance
}

// container3 implementation
type c3[T any, A1, A2 any, PA1 interface {
	*A1
	Init()
}, PA2 interface {
	*A2
	Init()
}, PT interface {
	*T
	Init(contracts.Container1[A1, PA1], contracts.Container1[A2, PA2])
}] struct {
	instance *T
	dep1     contracts.Container1[A1, PA1]
	dep2     contracts.Container1[A2, PA2]
}

func NewContainer3[T any, A1, A2 any, PA1 interface {
	*A1
	Init()
}, PA2 interface {
	*A2
	Init()
}, PT interface {
	*T
	Init(contracts.Container1[A1, PA1], contracts.Container1[A2, PA2])
}](arg1 contracts.Container1[A1, PA1], arg2 contracts.Container1[A2, PA2]) contracts.Container3[T, A1, A2, PA1, PA2, PT] {
	return &c3[T, A1, A2, PA1, PA2, PT]{dep1: arg1, dep2: arg2}
}

func (c *c3[T, A1, A2, PA1, PA2, PT]) Get() T {
	if c.instance == nil {
		var val T
		PT(&val).Init(c.dep1, c.dep2)
		c.instance = &val
	}
	return *c.instance
}

// container4 implementation
type c4[T any, A1, A2, A3 any, PA1 interface {
	*A1
	Init()
}, PA2 interface {
	*A2
	Init()
}, PA3 interface {
	*A3
	Init()
}, PT interface {
	*T
	Init(contracts.Container1[A1, PA1], contracts.Container1[A2, PA2], contracts.Container1[A3, PA3])
}] struct {
	instance *T
	dep1     contracts.Container1[A1, PA1]
	dep2     contracts.Container1[A2, PA2]
	dep3     contracts.Container1[A3, PA3]
}

func NewContainer4[T any, A1, A2, A3 any, PA1 interface {
	*A1
	Init()
}, PA2 interface {
	*A2
	Init()
}, PA3 interface {
	*A3
	Init()
}, PT interface {
	*T
	Init(contracts.Container1[A1, PA1], contracts.Container1[A2, PA2], contracts.Container1[A3, PA3])
}](arg1 contracts.Container1[A1, PA1], arg2 contracts.Container1[A2, PA2], arg3 contracts.Container1[A3, PA3]) contracts.Container4[T, A1, A2, A3, PA1, PA2, PA3, PT] {
	return &c4[T, A1, A2, A3, PA1, PA2, PA3, PT]{dep1: arg1, dep2: arg2, dep3: arg3}
}

func (c *c4[T, A1, A2, A3, PA1, PA2, PA3, PT]) Get() T {
	if c.instance == nil {
		var val T
		PT(&val).Init(c.dep1, c.dep2, c.dep3)
		c.instance = &val
	}
	return *c.instance
}

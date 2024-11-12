package stack

import "time"

const (
	Push Event = iota
	Pop
	Enqueue
	Peek
	Health
)

type Event int

type StackEvent[T any] struct {
	events    chan func() (Event, T, T)
	isPumping bool
}

func (se *StackEvent[T]) raise(e Event, current T, previous T) {
	se.events <- func() (Event, T, T) {
		return e, current, previous
	}
}

func (se *StackEvent[T]) subscribe(e Event) (T, T) {
	var current, previous T
	var event Event
	for eventFunc := range se.events {
		event, current, previous = eventFunc()
		if event == e {
			return current, previous
		}
		se.events <- eventFunc
	}
	return current, previous
}

func (se *StackEvent[T]) subscribeAsync(e Event, on func(current T, previous T)) {
	go (func() {
		current, previous := se.subscribe(e)
		on(current, previous)
	})()
}

func (se *StackEvent[T]) initialise() {
	if !se.isPumping {
		se.isPumping = true
		var current, previous T
		se.events <- func() (Event, T, T) {
			return Health, current, previous
		}
		for eFunc := range se.events {
			time.Sleep(1 * time.Second)
			se.events <- eFunc
		}
	}
}

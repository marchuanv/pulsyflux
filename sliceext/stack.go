package sliceext

type Stack[T comparable] struct {
	slice *slice[T]
}

func NewStack[T comparable]() Stack[T] {
	return Stack[T]{newSlice[T]()}
}

func (s *Stack[T]) Len() int {
	return s.slice.len()
}

func (s *Stack[T]) Push(item T) {
	s.slice.append(item)
}

func (s *Stack[T]) Pop() T {
	return s.slice.remove()
}

func (s *Stack[T]) Peek() T {
	return s.slice.peek()
}

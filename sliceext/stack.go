package sliceext

type Stack[T any] struct {
	slice *slice[T]
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{newSlice[T]()}
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

func (s *Stack[T]) Clone() *Stack[T] {
	stkCln := NewStack[T]()
	stkCln.slice = s.slice.copy()
	return stkCln
}

func (s *Stack[T]) Clear() {
	s.slice = newSlice[T]()
}

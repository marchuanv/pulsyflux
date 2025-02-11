package sliceext

type List[T comparable] struct {
	slice *slice[T]
}

func NewList[T comparable]() *List[T] {
	return &List[T]{newSlice[T]()}
}

func (l *List[T]) Len() int {
	return l.slice.len()
}

func (l *List[T]) Has(item T) bool {
	for index := 0; index < l.slice.len(); index++ {
		_item := l.slice.getAt(index)
		if _item == item {
			return true
		}
	}
	return false
}

func (l *List[T]) All() []T {
	arrCopy := make([]T, len(l.slice.arr))
	copy(arrCopy, l.slice.arr)
	return arrCopy
}

func (l *List[T]) Delete(item T) bool {
	for index := 0; index < l.slice.len(); index++ {
		itemAtIndex := l.slice.getAt(index)
		if itemAtIndex == item {
			l.slice.rmvAt(index)
			return true
		}
	}
	return false
}

func (l *List[T]) Add(item T) {
	l.slice.append(item)
}

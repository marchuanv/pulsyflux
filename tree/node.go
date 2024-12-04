package tree

import (
	"pulsyflux/sliceext"
	"sync"
)

type Node[T any] struct {
	mu       sync.Mutex
	ref      T
	parent   *Node[T]
	children sliceext.Stack[*Node[T]]
}

func NewNode[T any]() *Node[T] {
	var def T
	return &Node[T]{
		sync.Mutex{},
		def,
		nil,
		sliceext.NewStack[*Node[T]](),
	}
}

func (n *Node[T]) AddChild(childNode *Node[T]) {
	defer n.mu.Unlock()
	n.mu.Lock()
	n.children.Push(childNode)
	childNode.parent = n
}

func (n *Node[T]) GetLeafNode(isCriteriaMatch func(ref T) bool) *Node[T] {
	defer n.mu.Unlock()
	n.mu.Lock()
	child := n.children.ClonePop()
	for child != nil {
		found := child.GetLeafNode(isCriteriaMatch)
		if found != nil {
			return found
		}
		child = n.children.ClonePop()
	}
	if isCriteriaMatch(n.ref) {
		return n
	}
	return nil
}

func (n *Node[T]) Parent() *Node[T] {
	defer n.mu.Unlock()
	n.mu.Lock()
	return n.parent
}

func (n *Node[T]) Ref() T {
	defer n.mu.Unlock()
	n.mu.Lock()
	return n.ref
}

func (n *Node[T]) SetRef(ref T) {
	defer n.mu.Unlock()
	n.mu.Lock()
	n.ref = ref
}

package collections

import (
	"fmt"
	"sync"
)

type Dictionary[Key comparable, Value any] struct {
	mu sync.RWMutex
	m  map[Key]Value
}

func NewDictionary[Key comparable, Value any]() *Dictionary[Key, Value] {
	return &Dictionary[Key, Value]{
		m: make(map[Key]Value),
	}
}

func (d *Dictionary[Key, Value]) Len() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.m)
}

func (d *Dictionary[Key, Value]) Has(key Key) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.m[key]
	return ok
}

func (d *Dictionary[Key, Value]) Delete(key Key) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.m[key]; ok {
		delete(d.m, key)
		return true
	}
	return false
}

func (d *Dictionary[Key, Value]) Keys() []Key {
	d.mu.RLock()
	defer d.mu.RUnlock()
	keys := make([]Key, 0, len(d.m))
	for k := range d.m {
		keys = append(keys, k)
	}
	return keys
}

func (d *Dictionary[Key, Value]) Add(key Key, value Value) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.m[key]; ok {
		return fmt.Errorf("key already exists")
	}
	d.m[key] = value
	return nil
}

func (d *Dictionary[Key, Value]) Get(key Key) (Value, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	value, ok := d.m[key]
	return value, ok
}

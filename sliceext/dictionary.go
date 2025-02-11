package sliceext

type Dictionary[Key comparable, Value any] struct {
	keySlice   *slice[Key]
	valueSlice *slice[Value]
}

func NewDictionary[Key comparable, Value any]() *Dictionary[Key, Value] {
	return &Dictionary[Key, Value]{
		newSlice[Key](),
		newSlice[Value](),
	}
}

func (d *Dictionary[Key, Value]) Len() int {
	return d.keySlice.len()
}

func (d *Dictionary[Key, Value]) Has(key Key) bool {
	for index := 0; index < d.keySlice.len(); index++ {
		keyAtIndex := d.keySlice.getAt(index)
		if keyAtIndex == key {
			return true
		}
	}
	return false
}

func (d *Dictionary[Key, Value]) Delete(key Key) bool {
	for index := 0; index < d.keySlice.len(); index++ {
		keyAtIndex := d.keySlice.getAt(index)
		if keyAtIndex == key {
			d.keySlice.rmvAt(index)
			d.valueSlice.rmvAt(index)
			return true
		}
	}
	return false
}

func (d *Dictionary[Key, Value]) Keys(key Key) []Key {
	keys := make([]Key, d.keySlice.len())
	copy(keys, d.keySlice.arr)
	return keys
}

func (d *Dictionary[Key, Value]) Add(key Key, value Value) {
	if d.Has(key) {
		panic("call the Dictionary.Has() function to make sure that the key does not exist")
	}
	d.keySlice.append(key)
	d.valueSlice.append(value)
}

func (d *Dictionary[Key, Value]) Get(key Key) Value {
	var value Value
	for index := 0; index < d.keySlice.len(); index++ {
		keyAtIndex := d.keySlice.getAt(index)
		if keyAtIndex == key {
			value = d.valueSlice.getAt(index)
			break
		}
	}
	return value
}

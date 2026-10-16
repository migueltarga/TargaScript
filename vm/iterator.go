package vm

import (
	"fmt"

	"github.com/migueltarga/TargaScript/object"
)

// iterator produces the values of a repeat loop. Iterators live on the stack
// while their loop runs, but scripts never see them.
type iterator interface {
	object.Object

	// Next returns the next key and value. The key is the index for arrays,
	// strings and ranges, and the key for objects.
	Next() (key, value object.Object, ok bool)

	// KeysFirst reports whether a loop with one variable gets the keys, as
	// for objects, rather than the values.
	KeysFirst() bool
}

const ITERATOR_OBJ = "ITERATOR"

// newIterator returns an iterator over the characters of a string, the
// elements of an array, the keys and values of an object in insertion order,
// or 1..n for an integer n. Arrays and objects are iterated as they were
// when the loop started.
func newIterator(collection object.Object) (iterator, error) {
	switch c := collection.(type) {
	case *object.String:
		values := []object.Object{}
		for _, char := range c.Value {
			values = append(values, &object.String{Value: string(char)})
		}
		return &listIterator{values: values}, nil
	case *object.Array:
		return &listIterator{values: c.Elements}, nil
	case *object.Hash:
		it := &listIterator{keysFirst: true}
		for _, pair := range c.OrderedPairs() {
			it.keys = append(it.keys, pair.Key)
			it.values = append(it.values, pair.Value)
		}
		return it, nil
	case *object.Integer:
		return &rangeIterator{next: 1, end: c.Value}, nil
	}
	return nil, fmt.Errorf("cannot iterate over %s", collection.Type())
}

type listIterator struct {
	values    []object.Object
	keys      []object.Object // nil when the keys are the indexes
	keysFirst bool
	i         int
}

func (it *listIterator) Type() object.ObjectType { return ITERATOR_OBJ }
func (it *listIterator) Inspect() string         { return "iterator" }
func (it *listIterator) KeysFirst() bool         { return it.keysFirst }

func (it *listIterator) Next() (object.Object, object.Object, bool) {
	if it.i >= len(it.values) {
		return nil, nil, false
	}
	i := it.i
	it.i++
	if it.keys != nil {
		return it.keys[i], it.values[i], true
	}
	return &object.Integer{Value: int64(i)}, it.values[i], true
}

type rangeIterator struct {
	next, end int64
	index     int64
	done      bool
}

func (it *rangeIterator) Type() object.ObjectType { return ITERATOR_OBJ }
func (it *rangeIterator) Inspect() string         { return "iterator" }
func (it *rangeIterator) KeysFirst() bool         { return false }

func (it *rangeIterator) Next() (object.Object, object.Object, bool) {
	if it.done || it.next > it.end {
		return nil, nil, false
	}
	key := &object.Integer{Value: it.index}
	value := &object.Integer{Value: it.next}
	it.index++
	if it.next == it.end {
		// Stop here, as next+1 would overflow at the largest integer
		it.done = true
	} else {
		it.next++
	}
	return key, value, true
}

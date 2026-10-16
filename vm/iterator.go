package vm

import (
	"fmt"

	"github.com/migueltarga/TargaScript/object"
)

// iterator produces the values of a repeat loop. Iterators live on the stack
// while their loop runs, but scripts never see them.
type iterator interface {
	object.Object
	Next() (object.Object, bool)
}

const ITERATOR_OBJ = "ITERATOR"

// newIterator returns an iterator over the characters of a string, the
// elements of an array, the keys of an object in insertion order, or 1..n
// for an integer n. Arrays and objects are iterated as they were when the
// loop started.
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
		values := []object.Object{}
		for _, pair := range c.OrderedPairs() {
			values = append(values, pair.Key)
		}
		return &listIterator{values: values}, nil
	case *object.Integer:
		return &rangeIterator{next: 1, end: c.Value}, nil
	}
	return nil, fmt.Errorf("cannot iterate over %s", collection.Type())
}

type listIterator struct {
	values []object.Object
	i      int
}

func (it *listIterator) Type() object.ObjectType { return ITERATOR_OBJ }
func (it *listIterator) Inspect() string         { return "iterator" }

func (it *listIterator) Next() (object.Object, bool) {
	if it.i >= len(it.values) {
		return nil, false
	}
	it.i++
	return it.values[it.i-1], true
}

type rangeIterator struct {
	next, end int64
	done      bool
}

func (it *rangeIterator) Type() object.ObjectType { return ITERATOR_OBJ }
func (it *rangeIterator) Inspect() string         { return "iterator" }

func (it *rangeIterator) Next() (object.Object, bool) {
	if it.done || it.next > it.end {
		return nil, false
	}
	value := &object.Integer{Value: it.next}
	if it.next == it.end {
		// Stop here, as next+1 would overflow at the largest integer
		it.done = true
	} else {
		it.next++
	}
	return value, true
}

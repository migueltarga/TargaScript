package object

import "sync"

// Hash is an object: pairs in insertion order.
//
// Most objects have a few keys, and many objects have the same ones, like
// the {x, y} points of a program. Those share a Shape, which holds their
// keys, and keep only their values: less memory, and a shape the VM can
// remember where a key was in. Objects with many keys, or with a key
// deleted, are dictionaries instead: their own keys, and an index above a
// few.
type Hash struct {
	shape  *Shape   // nil for a dictionary
	values []Object // values[i] belongs to shape.keys[i]

	// A dictionary's pairs
	keys  []HashKey
	pairs []HashPair
	index map[HashKey]int
}

const (
	// indexAbove is the size above which a dictionary keeps an index
	indexAbove = 8
	// shapedUpTo is the most keys an object with a shape has
	shapedUpTo = 16
	// maxShapes limits the shapes made, as they are never freed; objects
	// that would need more are dictionaries
	maxShapes = 50_000
)

// Shape is the keys of objects that got the same keys in the same order.
// Shapes never change once made, so objects can share them.
type Shape struct {
	keys  []HashKey
	names []Object // the keys as values
	next  []transition
}

// transition leads from a shape to the one with a key more.
type transition struct {
	key   HashKey
	shape *Shape
}

var (
	emptyShape = &Shape{}
	// shapesMu guards the transitions of all shapes, as VMs on different
	// goroutines share them
	shapesMu   sync.Mutex
	shapeCount int
)

// Find returns the position of a key in the shape, or -1.
func (s *Shape) Find(key HashKey) int {
	for i := range s.keys {
		if s.keys[i] == key {
			return i
		}
	}
	return -1
}

// with returns the shape with a key added, or nil when no more shapes can
// be made.
func (s *Shape) with(key HashKey, name Object) *Shape {
	shapesMu.Lock()
	defer shapesMu.Unlock()
	for _, t := range s.next {
		if t.key == key {
			return t.shape
		}
	}
	if shapeCount >= maxShapes {
		return nil
	}
	shapeCount++
	next := &Shape{
		keys:  append(s.keys[:len(s.keys):len(s.keys)], key),
		names: append(s.names[:len(s.names):len(s.names)], name),
	}
	s.next = append(s.next, transition{key, next})
	return next
}

func NewHash() *Hash {
	return &Hash{shape: emptyShape}
}

// NewHashOfSize returns an empty hash with room for n pairs.
func NewHashOfSize(n int) *Hash {
	if n > shapedUpTo {
		return &Hash{}
	}
	return &Hash{shape: emptyShape, values: make([]Object, 0, n)}
}

// find returns the position of a key, or -1.
func (h *Hash) find(key HashKey) int {
	if h.shape != nil {
		return h.shape.Find(key)
	}
	if h.index != nil {
		if i, ok := h.index[key]; ok {
			return i
		}
		return -1
	}
	for i := range h.keys {
		if h.keys[i] == key {
			return i
		}
	}
	return -1
}

// Get returns the pair with the given key.
func (h *Hash) Get(key HashKey) (HashPair, bool) {
	if i := h.find(key); i >= 0 {
		_, pair := h.At(i)
		return pair, true
	}
	return HashPair{}, false
}

// Has reports whether the hash has the key.
func (h *Hash) Has(key HashKey) bool {
	return h.find(key) >= 0
}

// At returns the key and the pair at position i, in insertion order, for
// going through a hash without copying its pairs.
func (h *Hash) At(i int) (HashKey, HashPair) {
	if h.shape != nil {
		return h.shape.keys[i], HashPair{Key: h.shape.names[i], Value: h.values[i]}
	}
	return h.keys[i], h.pairs[i]
}

// Len returns the number of pairs.
func (h *Hash) Len() int {
	if h.shape != nil {
		return len(h.values)
	}
	return len(h.pairs)
}

// Set adds or updates a pair, keeping the original position of existing keys.
func (h *Hash) Set(key Hashable, value Object) {
	hashKey := key.HashKey()
	if i := h.find(hashKey); i >= 0 {
		if h.shape != nil {
			h.values[i] = value
		} else {
			h.pairs[i].Value = value
		}
		return
	}
	if h.shape != nil {
		if len(h.values) < shapedUpTo {
			if next := h.shape.with(hashKey, key.(Object)); next != nil {
				h.shape = next
				h.values = append(h.values, value)
				return
			}
		}
		h.toDictionary()
	}
	h.keys = append(h.keys, hashKey)
	h.pairs = append(h.pairs, HashPair{Key: key.(Object), Value: value})
	if h.index != nil {
		h.index[hashKey] = len(h.keys) - 1
	} else if len(h.keys) > indexAbove {
		h.reindex()
	}
}

// Delete removes a key, reporting whether it was there.
func (h *Hash) Delete(key Hashable) bool {
	i := h.find(key.HashKey())
	if i < 0 {
		return false
	}
	h.toDictionary()
	h.keys = append(h.keys[:i], h.keys[i+1:]...)
	h.pairs = append(h.pairs[:i], h.pairs[i+1:]...)
	if h.index != nil {
		h.reindex()
	}
	return true
}

// toDictionary gives a hash with a shape its own keys.
func (h *Hash) toDictionary() {
	if h.shape == nil {
		return
	}
	n := len(h.values)
	h.keys = make([]HashKey, n, n+1)
	h.pairs = make([]HashPair, n, n+1)
	for i := range h.values {
		h.keys[i], h.pairs[i] = h.At(i)
	}
	h.shape, h.values = nil, nil
	if n > indexAbove {
		h.reindex()
	}
}

func (h *Hash) reindex() {
	h.index = make(map[HashKey]int, len(h.keys))
	for i, key := range h.keys {
		h.index[key] = i
	}
}

// OrderedPairs returns a copy of the pairs in insertion order.
func (h *Hash) OrderedPairs() []HashPair {
	pairs := make([]HashPair, h.Len())
	for i := range pairs {
		_, pairs[i] = h.At(i)
	}
	return pairs
}

// PropertyCache remembers, for one place in a program that reads a property,
// the shape of the last object read there and where the key was in it. The
// next object of the same shape needs no search.
type PropertyCache struct {
	shape *Shape
	index int
}

// Lookup returns the value of a key, using and updating cache.
func (h *Hash) Lookup(key HashKey, cache *PropertyCache) (Object, bool) {
	if h.shape != nil {
		if h.shape == cache.shape {
			return h.values[cache.index], true
		}
		if i := h.shape.Find(key); i >= 0 {
			cache.shape, cache.index = h.shape, i
			return h.values[i], true
		}
		return nil, false
	}
	pair, ok := h.Get(key)
	return pair.Value, ok
}

// Update replaces the value of a key the hash has, using and updating
// cache, and reports whether it had the key.
func (h *Hash) Update(key HashKey, cache *PropertyCache, value Object) bool {
	if h.shape != nil {
		if h.shape == cache.shape {
			h.values[cache.index] = value
			return true
		}
		if i := h.shape.Find(key); i >= 0 {
			cache.shape, cache.index = h.shape, i
			h.values[i] = value
			return true
		}
		return false
	}
	if i := h.find(key); i >= 0 {
		h.pairs[i].Value = value
		return true
	}
	return false
}

// NewHashFromPairs makes the object of an object literal from its keys and
// values, one after the other in pairs. The cache of the literal remembers
// the shape of the object it made, so the next one starts with it instead
// of adding its keys one by one. It returns the position of a key that
// can't be one, or -1.
func NewHashFromPairs(pairs []Object, cache *PropertyCache) (*Hash, int) {
	n := len(pairs) / 2
	if s := cache.shape; s != nil && len(s.keys) == n {
		values := make([]Object, n)
		same := true
		for i := 0; i < n && same; i++ {
			key, ok := pairs[2*i].(Hashable)
			same = ok && key.HashKey() == s.keys[i]
			values[i] = pairs[2*i+1]
		}
		if same {
			return &Hash{shape: s, values: values}, -1
		}
	}

	h := NewHashOfSize(n)
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(Hashable)
		if !ok {
			return nil, i
		}
		h.Set(key, pairs[i+1])
	}
	if h.shape != nil {
		cache.shape = h.shape
	}
	return h, -1
}

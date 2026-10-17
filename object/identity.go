package object

import "unsafe"

// Identity is what tells a value apart from every other: its type and its
// address, as every object is a pointer. Comparing objects with == or using
// them as map keys would make TinyGo compare and hash them with reflection,
// which adds kilobytes to the WebAssembly build; identities compare and hash
// as plain numbers. As numbers they don't keep the value alive, so a map
// keyed by them holds only values kept somewhere else too.
type Identity [2]uintptr

// IdentityOf returns the identity of obj.
func IdentityOf(obj Object) Identity {
	return *(*Identity)(unsafe.Pointer(&obj))
}

// Same reports whether a and b are the same value, as a == b does.
func Same(a, b Object) bool {
	return IdentityOf(a) == IdentityOf(b)
}

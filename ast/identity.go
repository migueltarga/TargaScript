package ast

import "unsafe"

// Identity tells a node apart from every other, by its type and address,
// for maps keyed by nodes. A map keyed by Node itself would make TinyGo hash
// interfaces with reflection, which adds kilobytes to the WebAssembly build.
// As numbers, identities don't keep the nodes alive.
type Identity [2]uintptr

// IdentityOf returns the identity of node.
func IdentityOf(node Node) Identity {
	return *(*Identity)(unsafe.Pointer(&node))
}

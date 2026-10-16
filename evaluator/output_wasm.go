//go:build tinygo && wasm

package evaluator

import "io"

// The browser build sets the output when it starts, and leaves out the os
// package
var defaultOutput io.Writer = io.Discard

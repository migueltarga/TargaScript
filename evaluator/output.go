//go:build !(tinygo && wasm)

package evaluator

import (
	"io"
	"os"
)

var defaultOutput io.Writer = os.Stdout

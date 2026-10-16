//go:build !(tinygo && wasm)

package compiler

import (
	"os"
	"path/filepath"
)

// ReadFile reads the files programs load. The browser build reads them from
// the page's server instead.
var ReadFile = os.ReadFile

var (
	isNotExist = os.IsNotExist
	isAbs      = filepath.IsAbs
	joinPath   = filepath.Join
	absPath    = filepath.Abs
	dirOf      = filepath.Dir
)

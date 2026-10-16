//go:build tinygo && wasm

package compiler

import (
	"errors"
	"path"
)

// The browser build has no files of its own, and leaves out the os package
// with what it pulls in, like the time package.

// ReadFile reads the files programs load; the browser sets it to read them
// from the page's server. It returns ErrNotExist for a file that isn't
// there.
var ReadFile = func(string) ([]byte, error) { return nil, ErrNotExist }

// ErrNotExist is the error of ReadFile for a missing file.
var ErrNotExist = errors.New("file not found")

func isNotExist(err error) bool { return err == ErrNotExist }

var (
	isAbs    = path.IsAbs
	joinPath = path.Join
	dirOf    = path.Dir
)

func absPath(file string) (string, error) { return path.Clean(file), nil }

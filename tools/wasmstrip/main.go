// Command wasmstrip removes the custom sections of a WebAssembly file, such
// as function names and the compiler's producer information, which the
// browser doesn't need.
//
// usage: go run ./tools/wasmstrip file.wasm
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: wasmstrip file.wasm")
		os.Exit(2)
	}
	path := os.Args[1]

	input, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	output, err := strip(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, output, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s: %d -> %d bytes\n", path, len(input), len(output))
}

var magic = []byte{0x00, 'a', 's', 'm', 0x01, 0x00, 0x00, 0x00}

// strip copies every section except custom ones (id 0).
func strip(input []byte) ([]byte, error) {
	if !bytes.HasPrefix(input, magic) {
		return nil, errors.New("not a WebAssembly module")
	}

	output := append([]byte{}, magic...)
	for rest := input[len(magic):]; len(rest) > 0; {
		id := rest[0]
		size, n := uleb128(rest[1:])
		if n == 0 || uint64(len(rest)-1-n) < size {
			return nil, errors.New("truncated section")
		}
		end := 1 + n + int(size)
		if id != 0 {
			output = append(output, rest[:end]...)
		}
		rest = rest[end:]
	}
	return output, nil
}

// uleb128 decodes an unsigned LEB128 number, returning it and the number of
// bytes read, or 0 bytes if b is too short.
func uleb128(b []byte) (uint64, int) {
	var value uint64
	for i := 0; i < len(b) && i < 10; i++ {
		value |= uint64(b[i]&0x7f) << (7 * i)
		if b[i]&0x80 == 0 {
			return value, i + 1
		}
	}
	return 0, 0
}

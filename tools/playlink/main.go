// Command playlink prints a link that opens a program in the playground,
// the same kind of link its Share button makes. It reads the program from
// the file given, or from standard input.
//
//	go run ./tools/playlink examples/fibonacci.tg
package main

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
)

// Playground is the address of the playground.
const Playground = "https://migueltarga.github.io/TargaScript/"

// Link returns the playground link for a program: the code compressed with
// raw deflate and written in base64url without padding, after #code=, as
// the playground's CompressionStream("deflate-raw") does.
func Link(code string) string {
	var compressed bytes.Buffer
	w, _ := flate.NewWriter(&compressed, flate.BestCompression)
	io.WriteString(w, code)
	w.Close()
	return Playground + "#code=" + base64.RawURLEncoding.EncodeToString(compressed.Bytes())
}

// Code returns the program in a playground link.
func Code(link string) (string, error) {
	_, encoded, ok := strings.Cut(link, "#code=")
	if !ok {
		return "", fmt.Errorf("not a playground link with code: %s", link)
	}
	compressed, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	code, err := io.ReadAll(flate.NewReader(bytes.NewReader(compressed)))
	return string(code), err
}

func main() {
	var code []byte
	var err error
	if len(os.Args) > 1 {
		code, err = os.ReadFile(os.Args[1])
	} else {
		code, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(Link(string(code)))
}

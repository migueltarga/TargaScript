package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/token"
)

// bundleMagic ends an executable made by targascript build. Before it come
// the bundle, as JSON, and its length.
const bundleMagic = "TGBUNDLE"

// bundle is a program and the files it loads, by their absolute paths.
type bundle struct {
	Main  string            `json:"main"`
	Files map[string]string `json:"files"`
}

// runBuild makes an executable that runs the program in file, with the
// files it loads, so it runs where TargaScript isn't installed. It returns
// the exit code.
func runBuild(args []string) int {
	file, output := "", ""
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "-o" && i+1 < len(args):
			output = args[i+1]
			i++
		case file == "":
			file = args[i]
		default:
			fmt.Fprintln(os.Stderr, "usage: targascript build file.tg [-o name]")
			return 1
		}
	}
	if file == "" {
		fmt.Fprintln(os.Stderr, "usage: targascript build file.tg [-o name]")
		return 1
	}
	if output == "" {
		output = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		if runtime.GOOS == "windows" {
			output += ".exe"
		}
	}

	main, err := filepath.Abs(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	b := bundle{Main: main, Files: map[string]string{}}
	if err := b.add(main); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	self, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	runtimeBinary, err := os.ReadFile(self)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	// An executable that was built itself carries a bundle already
	if size, ok := bundleSize(runtimeBinary); ok {
		runtimeBinary = runtimeBinary[:len(runtimeBinary)-size]
	}

	payload, err := json.Marshal(b)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	var out bytes.Buffer
	out.Write(runtimeBinary)
	out.Write(payload)
	binary.Write(&out, binary.LittleEndian, uint64(len(payload)))
	out.WriteString(bundleMagic)
	if err := os.WriteFile(output, out.Bytes(), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("built %s (%d files, %.1f MB)\n", output, len(b.Files), float64(out.Len())/(1<<20))
	return 0
}

// add reads a file into the bundle with the files it loads.
func (b *bundle) add(path string) error {
	if _, done := b.Files[path]; done {
		return nil
	}
	source, err := compiler.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", path, err)
	}
	b.Files[path] = string(source)

	// Every load "path" in the file, wherever it is
	l := lexer.New(string(source))
	previous := token.Token{}
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		if previous.Type == token.LOAD && tok.Type == token.STRING {
			loaded, err := compiler.ResolveLoadPath(compiler.ModuleDir(path), tok.Literal)
			if err != nil {
				return err
			}
			if err := b.add(loaded); err != nil {
				return err
			}
		}
		previous = tok
	}
	return nil
}

// bundleSize returns the size of the bundle at the end of an executable,
// its length and magic included.
func bundleSize(executable []byte) (int, bool) {
	tail := len(bundleMagic) + 8
	if len(executable) < tail || string(executable[len(executable)-len(bundleMagic):]) != bundleMagic {
		return 0, false
	}
	size := binary.LittleEndian.Uint64(executable[len(executable)-tail:])
	if size > uint64(len(executable)-tail) {
		return 0, false
	}
	return int(size) + tail, true
}

// readBundle returns the bundle of the running executable, if it has one.
func readBundle() (*bundle, bool) {
	self, err := os.Executable()
	if err != nil {
		return nil, false
	}
	file, err := os.Open(self)
	if err != nil {
		return nil, false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, false
	}

	tail := make([]byte, len(bundleMagic)+8)
	if info.Size() < int64(len(tail)) {
		return nil, false
	}
	if _, err := file.ReadAt(tail, info.Size()-int64(len(tail))); err != nil || string(tail[8:]) != bundleMagic {
		return nil, false
	}
	size := int64(binary.LittleEndian.Uint64(tail))
	if size > info.Size()-int64(len(tail)) {
		return nil, false
	}
	payload := make([]byte, size)
	if _, err := file.ReadAt(payload, info.Size()-int64(len(tail))-size); err != nil {
		return nil, false
	}
	b := &bundle{}
	if json.Unmarshal(payload, b) != nil {
		return nil, false
	}
	return b, true
}

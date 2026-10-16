package main

import (
	"fmt"
	"os"

	"github.com/migueltarga/TargaScript/format"
)

// runFmt lays out the .tg files in paths, or under the current directory,
// in place. With --check it changes nothing and fails when a file isn't
// laid out, for CI.
func runFmt(args []string) int {
	check := false
	paths := []string{}
	for _, arg := range args {
		if arg == "--check" {
			check = true
			continue
		}
		paths = append(paths, arg)
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	files, err := sourceFiles(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	changed := 0
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		formatted := format.Source(string(source))
		if formatted == string(source) {
			continue
		}
		changed++
		if check {
			fmt.Println(file)
			continue
		}
		if err := os.WriteFile(file, []byte(formatted), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("formatted", file)
	}
	if check && changed > 0 {
		fmt.Printf("%d of %d files need targascript fmt\n", changed, len(files))
		return 1
	}
	return 0
}

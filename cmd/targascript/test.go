package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/repl"
)

// runTests runs the test files in paths, or under the current directory
// when there are none, and returns the exit code: 1 when a test or a file
// failed, or when there were no tests.
func runTests(paths []string, noColor bool) int {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	files, err := testFiles(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no test files found: test files end in _test.tg")
		return 1
	}

	failedFiles := 0
	for _, file := range files {
		fmt.Println(file)
		input, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			failedFiles++
			continue
		}
		if abs, err := filepath.Abs(file); err == nil {
			repl.SetMainDir(filepath.Dir(abs))
		}
		// An error outside a test stops the file
		if executeFile(string(input), false, noColor) != nil {
			failedFiles++
		}
		fmt.Println()
	}

	passed, failed := evaluator.TestResults()
	summary := fmt.Sprintf("%d passed, %d failed", passed, failed)
	if failedFiles > 0 {
		summary += fmt.Sprintf(", %d file(s) with errors", failedFiles)
	}
	fmt.Println(summary)
	if failed > 0 || failedFiles > 0 || passed == 0 {
		return 1
	}
	return 0
}

// testFiles lists the files to run: files given by name, and the files
// ending in _test.tg under the directories given.
func testFiles(paths []string) ([]string, error) {
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			files = append(files, path)
			continue
		}
		err = filepath.WalkDir(path, func(file string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() && file != path && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			if !entry.IsDir() && strings.HasSuffix(file, "_test.tg") {
				files = append(files, file)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

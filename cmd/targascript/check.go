package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/migueltarga/TargaScript/diag"
	"github.com/migueltarga/TargaScript/repl"
)

// runCheck checks the .tg files in paths, or under the current directory
// when there are none, without running them, and returns the exit code: 1
// when a file has a problem.
func runCheck(paths []string) int {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	files, err := sourceFiles(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	failed := 0
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed++
			continue
		}
		dir := filepath.Dir(file)
		if abs, err := filepath.Abs(dir); err == nil {
			dir = abs
		}
		if problems := repl.Check(string(source), dir); len(problems) > 0 {
			failed++
			fmt.Printf("%s\n%s\n\n", file, diag.FormatAll(string(source), problems))
		}
	}

	if failed > 0 {
		fmt.Printf("%d of %d files have problems\n", failed, len(files))
		return 1
	}
	fmt.Printf("%d files checked, no problems\n", len(files))
	return 0
}

// sourceFiles lists the files named in paths and the .tg files under the
// directories, skipping hidden ones.
func sourceFiles(paths []string) ([]string, error) {
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
			if !entry.IsDir() && strings.HasSuffix(file, ".tg") {
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

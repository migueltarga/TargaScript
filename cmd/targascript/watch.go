package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// watchInterval is how often watch looks for changed files.
const watchInterval = 300 * time.Millisecond

// runWatch runs targascript with args, and again whenever a .tg file under
// dir changes, stopping the run in progress. Each run is a process of its
// own, so exit and endless loops don't stop the watching. It returns only
// when the files can't be read.
func runWatch(args []string, dir string) int {
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	last, err := snapshot(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	for {
		cmd := exec.Command(self, args...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		done := make(chan error, 1)
		if err := cmd.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		go func() { done <- cmd.Wait() }()

		running := true
		for {
			select {
			case <-done:
				running = false
				fmt.Fprintln(os.Stderr, "\033[2m[watching for changes, Ctrl+C to stop]\033[0m")
				done = nil
			case <-time.After(watchInterval):
			}
			now, err := snapshot(dir)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			if now != last {
				last = now
				break
			}
		}
		if running {
			cmd.Process.Kill()
			<-done
		}
		fmt.Fprint(os.Stderr, "\033[2m[changed, running again]\033[0m\n\n")
	}
}

// snapshot sums up the .tg files under dir: their names, sizes and times,
// so that any change gives a different one.
func snapshot(dir string) (string, error) {
	var out strings.Builder
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != dir && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".tg" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil // removed while looking
		}
		fmt.Fprintf(&out, "%s %d %d\n", path, info.Size(), info.ModTime().UnixNano())
		return nil
	})
	return out.String(), err
}

package repl

import (
	"os"
	"sort"
)

// osSystem gives programs run from the command line access to files, the
// environment and the exit code.
type osSystem struct{}

func (osSystem) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}

func (osSystem) WriteFile(path, content string, appendToFile bool) error {
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if appendToFile {
		flags = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.WriteString(content); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func (osSystem) ListDir(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	sort.Strings(names)
	return names, nil
}

func (osSystem) Env(name string) (string, bool) {
	return os.LookupEnv(name)
}

func (osSystem) Exit(code int) {
	os.Exit(code)
}

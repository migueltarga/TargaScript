package repl

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/migueltarga/TargaScript/compiler"
)

// moduleClient downloads files loaded from web addresses.
var moduleClient = &http.Client{Timeout: 30 * time.Second}

// downloaded keeps the files loaded from web addresses, which the compiler
// and the VM both read, so each is downloaded once per run.
var downloaded = map[string][]byte{}

// readModule reads a loaded file from disk, or downloads it when its path is
// a web address.
func readModule(path string) ([]byte, error) {
	if !compiler.IsURL(path) {
		return os.ReadFile(path)
	}
	if source, ok := downloaded[path]; ok {
		return source, nil
	}
	response, err := moduleClient.Get(path)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the server answered %s", response.Status)
	}
	source, err := io.ReadAll(io.LimitReader(response.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	downloaded[path] = source
	return source, nil
}

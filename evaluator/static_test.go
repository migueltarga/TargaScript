package evaluator

import (
	"errors"
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

// fakeFiles is a System with files in memory; a directory is any prefix
// of a file's path.
type fakeFiles map[string]string

func (f fakeFiles) ReadFile(path string) (string, error) {
	if content, ok := f[path]; ok {
		return content, nil
	}
	return "", errors.New("no such file")
}
func (f fakeFiles) WriteFile(path, content string, appendToFile bool) error { return nil }
func (f fakeFiles) ListDir(path string) ([]string, error) {
	for name := range f {
		if strings.HasPrefix(name, path+"/") {
			return nil, nil
		}
	}
	return nil, errors.New("not a directory")
}
func (f fakeFiles) Env(name string) (string, bool) { return "", false }
func (f fakeFiles) Exit(code int)                  {}

func TestStaticFile(t *testing.T) {
	SetSystem(fakeFiles{
		"site/index.html":      "<h1>Home</h1>",
		"site/app.js":          "run()",
		"site/logo.PNG":        "png",
		"site/data.bin":        "bin",
		"site/docs/index.html": "<p>docs</p>",
		"site/.env":            "secret",
		"secret.txt":           "outside",
	})
	defer SetSystem(nil)

	tests := []struct {
		path        string
		status      int
		contentType string
		body        string
	}{
		{"/", 200, "text/html; charset=utf-8", "<h1>Home</h1>"},
		{"/app.js", 200, "text/javascript; charset=utf-8", "run()"},
		{"/logo.PNG", 200, "image/png", "png"},
		{"/data.bin", 200, "application/octet-stream", "bin"},
		{"/docs/", 200, "text/html; charset=utf-8", "<p>docs</p>"},
		{"/docs", 301, "", ""},
		{"/missing.html", 404, "text/plain; charset=utf-8", "Not found\n"},
		{"/.env", 404, "text/plain; charset=utf-8", "Not found\n"},
		{"/../secret.txt", 404, "text/plain; charset=utf-8", "Not found\n"},
		{"/docs/../../secret.txt", 404, "text/plain; charset=utf-8", "Not found\n"},
	}
	for _, tt := range tests {
		result := staticFileBuiltin(&object.String{Value: "site"}, &object.String{Value: tt.path})
		status, headers, body := Reply(result)
		if status != tt.status || headers["Content-Type"] != tt.contentType || body != tt.body {
			t.Errorf("%s: got %d %q %q, want %d %q %q", tt.path, status, headers["Content-Type"], body, tt.status, tt.contentType, tt.body)
		}
	}

	_, headers, _ := Reply(staticFileBuiltin(&object.String{Value: "site"}, &object.String{Value: "/docs"}))
	if headers["Location"] != "/docs/" {
		t.Errorf("/docs redirects to %q, want /docs/", headers["Location"])
	}
}

func TestStaticFileWithoutSystem(t *testing.T) {
	SetSystem(nil)
	result := staticFileBuiltin(&object.String{Value: "site"}, &object.String{Value: "/"})
	if err, ok := result.(*object.Error); !ok || err.Message != "staticFile is only available in the command line" {
		t.Errorf("got %s", result.Inspect())
	}
}

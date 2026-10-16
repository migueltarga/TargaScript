package repl

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repo":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"name": "TargaScript", "stars": 42}`)
		case "/echo":
			body, _ := io.ReadAll(r.Body)
			fmt.Fprintf(w, "%s %s %s %s", r.Method, r.Header.Get("Content-Type"), r.Header.Get("X-Token"), body)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tests := []struct {
		program  string
		expected string
	}{
		{`let r = await fetch("URL/repo")
print(r.status, r.ok, r.headers["content-type"])
let repo = r.json()
print(repo.name, repo.stars)`, "200 true application/json\nTargaScript 42\n"},
		{`let r = await fetch("URL/echo", {"method": "post", "headers": {"X-Token": "abc"}, "body": {"a": [1, 2]}})
print(r.text)`, "POST application/json abc {\"a\":[1,2]}\n"},
		{`let r = await fetch("URL/missing")
print(r.status, r.ok)`, "404 false\n"},
		{`let results = []
repeat path in ["repo", "echo"] {
    let r = await fetch("URL/" + path)
    results.insert(r.status)
}
print(results)`, "[200, 200]\n"},
		{`print(try { await fetch("http://127.0.0.1:1/") } catch e { e.message.startsWith("fetch http://127.0.0.1:1/ failed") })`, "true\n"},
		{`print(try { fetch("URL", {"color": 1}) } catch e { e.message })`, "unknown fetch option \"color\", expected method, headers or body\n"},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		program := strings.ReplaceAll(tt.program, "URL", server.URL)
		if err := EvalSource(program, &out, true, false, false); err != nil {
			t.Errorf("%s\nerror: %v", tt.program, err)
			continue
		}
		if out.String() != tt.expected {
			t.Errorf("%s\n got: %q\nwant: %q", tt.program, out.String(), tt.expected)
		}
	}
}

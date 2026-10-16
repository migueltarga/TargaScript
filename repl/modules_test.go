package repl

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoadFromURL(t *testing.T) {
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		switch r.URL.Path {
		case "/lib/main.tg":
			fmt.Fprint(w, "load \"./helper.tg\"\nfn shout(t) { exclaim(t.upper()) }")
		case "/lib/helper.tg":
			fmt.Fprint(w, "fn exclaim(t) { t + \"!\" }")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var out bytes.Buffer
	program := fmt.Sprintf("load %q as lib\nprint(lib.shout(\"hi\"))", server.URL+"/lib/main.tg")
	if err := EvalSource(program, &out, true, false, false); err != nil {
		t.Fatal(err)
	}
	if out.String() != "HI!\n" {
		t.Errorf("printed %q", out.String())
	}
	// The compiler and the VM both read it, but it's downloaded once
	if requests["/lib/main.tg"] != 1 || requests["/lib/helper.tg"] != 1 {
		t.Errorf("requests: %v", requests)
	}

	err := EvalSource(fmt.Sprintf("load %q", server.URL+"/missing.tg"), &out, true, false, false)
	if err == nil || !strings.Contains(err.Error(), "the server answered 404") {
		t.Errorf("missing file: %v", err)
	}
}

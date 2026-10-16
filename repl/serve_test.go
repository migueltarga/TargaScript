package repl

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/eventloop"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

// handlerServer starts a test server with a TargaScript handler, the value
// of source. The handler runs on an event loop of its own, which runs until
// the test ends.
func handlerServer(t *testing.T, source string) *httptest.Server {
	t.Helper()
	l := eventloop.New()
	evaluator.SetHost(l)
	t.Cleanup(func() { evaluator.SetHost(loop) })

	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	handler := newSession().run(program)
	if !evaluator.IsCallable(handler) {
		t.Fatalf("the source isn't a function: %s", handler.Inspect())
	}

	var stopped atomic.Bool
	done := make(chan bool)
	l.Hold()
	go func() {
		l.Run(stopped.Load)
		close(done)
	}()
	t.Cleanup(func() {
		stopped.Store(true)
		l.Post(func() {})
		<-done
	})

	server := httptest.NewServer(newHandler(l.Post, func(request *object.Hash) object.Object {
		return evaluator.CallFunction(handler, []object.Object{request})
	}))
	t.Cleanup(server.Close)
	return server
}

func TestServe(t *testing.T) {
	var log bytes.Buffer
	evaluator.SetOutput(&log)
	t.Cleanup(func() { evaluator.SetOutput(os.Stdout) })

	server := handlerServer(t, `
let items = ["a"]
fn(req) {
    match req.path {
        "/" => "<h1>Hi</h1>"
        "/text" => "plain"
        "/items" => items
        "/hello" => "Hello, ${req.query.name ?? "world"} from ${req.headers["user-agent"]}"
        "/add" => {
            items.insert(parseJSON(req.body).name)
            response({count: items.length}, {status: 201, headers: {"X-Count": items.length}})
        }
        "/empty" => null
        "/later" => {
            await sleep(5)
            "awaited"
        }
        "/rejected" => {
            await sleep(1)
            throw "too late"
        }
        "/fail" => null.x
        _ => response("Not found", {status: 404})
    }
}`)

	tests := []struct {
		method, path, body string
		status             int
		contentType, reply string
	}{
		{"GET", "/", "", 200, "text/html; charset=utf-8", "<h1>Hi</h1>"},
		{"GET", "/text", "", 200, "text/plain; charset=utf-8", "plain"},
		{"GET", "/items", "", 200, "application/json", `["a"]`},
		{"GET", "/hello?name=Ana", "", 200, "text/plain; charset=utf-8", "Hello, Ana from test"},
		{"POST", "/add", `{"name": "b"}`, 201, "application/json", `{"count":2}`},
		{"GET", "/items", "", 200, "application/json", `["a","b"]`},
		{"GET", "/empty", "", 204, "", ""},
		{"GET", "/later", "", 200, "text/plain; charset=utf-8", "awaited"},
		{"GET", "/rejected", "", 500, "text/plain; charset=utf-8", "too late\n"},
		{"GET", "/fail", "", 500, "text/plain; charset=utf-8", "cannot read property \"x\" of null. To read it only when there's a value, write ?.x\n"},
		{"GET", "/missing", "", 404, "text/plain; charset=utf-8", "Not found"},
	}
	for _, tt := range tests {
		req, _ := http.NewRequest(tt.method, server.URL+tt.path, strings.NewReader(tt.body))
		req.Header.Set("User-Agent", "test")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != tt.status || res.Header.Get("Content-Type") != tt.contentType || string(body) != tt.reply {
			t.Errorf("%s %s: got %d %q %q, want %d %q %q", tt.method, tt.path,
				res.StatusCode, res.Header.Get("Content-Type"), body, tt.status, tt.contentType, tt.reply)
		}
		if tt.path == "/add" && res.Header.Get("X-Count") != "2" {
			t.Errorf("X-Count = %q, want 2", res.Header.Get("X-Count"))
		}
	}

	if !strings.Contains(log.String(), `GET /fail: ERROR at line 22`) {
		t.Errorf("the failing request wasn't logged: %q", log.String())
	}
}

func TestServeErrors(t *testing.T) {
	tests := []struct{ program, expected string }{
		{`serve(8080)`, "wrong number of arguments for serve"},
		{`serve(70000, fn(req) { "" })`, "must be an integer from 0 to 65535"},
		{`serve(8080, "hi")`, "must be a function, got a string"},
		{`response("x", {code: 1})`, `response has no option "code"`},
		{`response("x", {status: 42})`, "from 100 to 999"},
	}
	for _, tt := range tests {
		var out bytes.Buffer
		err := EvalSource(tt.program, &out, true, false, false)
		if err == nil || !strings.Contains(err.Error(), tt.expected) {
			t.Errorf("%s: got %v, want an error with %q", tt.program, err, tt.expected)
		}
	}
}

func TestServeAnswersWhileAHandlerWaits(t *testing.T) {
	server := handlerServer(t, `
fn(req) {
    if req.path == "/slow" {
        await sleep(300)
        return "slow"
    }
    "fast"
}`)

	slow := make(chan string)
	go func() {
		res, err := http.Get(server.URL + "/slow")
		if err != nil {
			slow <- err.Error()
			return
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		slow <- string(body)
	}()

	start := time.Now()
	res, err := http.Get(server.URL + "/fast")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if string(body) != "fast" || time.Since(start) > 200*time.Millisecond {
		t.Errorf("/fast answered %q after %s, while /slow was waiting", body, time.Since(start))
	}
	if got := <-slow; got != "slow" {
		t.Errorf("/slow answered %q", got)
	}
}

package repl

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/object"
)

// maxRequestBody limits the body a handler can receive.
const maxRequestBody = 10 << 20

// httpServer serves HTTP for serve with net/http.
type httpServer struct{}

func (httpServer) Serve(port int, handle func(request *object.Hash) object.Object) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	fmt.Fprintf(evaluator.Output(), "Listening on http://localhost:%d\n", listener.Addr().(*net.TCPAddr).Port)
	return http.Serve(listener, newHandler(handle))
}

// newHandler calls handle for each request, one at a time, since programs
// run on a single thread.
func newHandler(handle func(request *object.Hash) object.Object) http.Handler {
	var mu sync.Mutex
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBody))
		if err != nil {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		mu.Lock()
		result := handle(requestObject(r, string(body)))
		status, headers, text := evaluator.Reply(result)
		mu.Unlock()

		if err, ok := result.(*object.Error); ok {
			fmt.Fprintf(evaluator.Output(), "%s %s: %s\n", r.Method, r.URL.Path, err.Inspect())
		}
		for name, value := range headers {
			w.Header().Set(name, value)
		}
		w.WriteHeader(status)
		io.WriteString(w, text)
	})
}

// requestObject is the object a handler receives: method, path, query,
// headers (with lowercase names) and body.
func requestObject(r *http.Request, body string) *object.Hash {
	str := func(s string) *object.String { return &object.String{Value: s} }

	query := object.NewHash()
	for name, values := range r.URL.Query() {
		query.Set(str(name), str(values[0]))
	}
	headers := object.NewHash()
	for name, values := range r.Header {
		headers.Set(str(strings.ToLower(name)), str(strings.Join(values, ", ")))
	}

	request := object.NewHash()
	request.Set(str("method"), str(r.Method))
	request.Set(str("path"), str(r.URL.Path))
	request.Set(str("query"), query)
	request.Set(str("headers"), headers)
	request.Set(str("body"), str(body))
	return request
}

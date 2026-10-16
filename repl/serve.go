package repl

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/eventloop"
	"github.com/migueltarga/TargaScript/object"
)

// maxRequestBody limits the body a handler can receive.
const maxRequestBody = 10 << 20

// httpServer serves HTTP for serve with net/http. Handlers run on the event
// loop, so they can await, and the loop keeps the program running.
type httpServer struct {
	loop *eventloop.Loop
}

func (s httpServer) Serve(port int, handle func(request *object.Hash) object.Object) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	fmt.Fprintf(evaluator.Output(), "Listening on http://localhost:%d\n", listener.Addr().(*net.TCPAddr).Port)
	s.loop.Hold()
	go http.Serve(listener, newHandler(s.loop.Post, handle))
	return nil
}

// reply is what a handler's result becomes.
type reply struct {
	status  int
	headers map[string]string
	body    string
}

// newHandler passes each request to handle on the loop, through post, and
// writes what it returns, waiting for it when it's a promise.
func newHandler(post func(func()), handle func(request *object.Hash) object.Object) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBody))
		if err != nil {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		replies := make(chan reply, 1)
		post(func() {
			finish := func(result object.Object) {
				if err, ok := result.(*object.Error); ok {
					fmt.Fprintf(evaluator.Output(), "%s %s: %s\n", r.Method, r.URL.Path, err.Inspect())
				}
				status, headers, text := evaluator.Reply(result)
				replies <- reply{status, headers, text}
			}

			result := handle(requestObject(r, string(body)))
			promise, ok := result.(*object.Promise)
			if !ok {
				finish(result)
				return
			}
			promise.Then(finish, func(reason object.Object) {
				if _, ok := reason.(*object.Error); !ok {
					reason = &object.Error{Message: evaluator.ToDisplayString(reason)}
				}
				finish(reason)
			})
		})

		reply := <-replies
		for name, value := range reply.headers {
			w.Header().Set(name, value)
		}
		w.WriteHeader(reply.status)
		io.WriteString(w, reply.body)
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

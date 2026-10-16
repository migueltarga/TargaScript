package evaluator

import (
	"strings"

	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/strfmt"
)

// Server runs the HTTP server of serve. Only the command line sets one.
type Server interface {
	// Serve starts listening on port and returns. It then calls handle for
	// each request, one at a time, on the goroutine that runs the program.
	// handle can return a promise, when the handler awaits.
	Serve(port int, handle func(request *object.Hash) object.Object) error
}

var server Server

// SetServer sets what serve uses.
func SetServer(s Server) {
	server = s
}

// Response is what response() returns: a body with a status and headers.
type Response struct {
	Status  int
	Headers map[string]string
	Body    object.Object
}

func (r *Response) Type() object.ObjectType { return "RESPONSE" }
func (r *Response) Inspect() string {
	return strfmt.Sprintf("response(%d, %s)", r.Status, r.Body.Inspect())
}

// Reply turns what a handler returned into the status, headers and body to
// send: strings as they are, null as an empty 204, and other values as
// JSON. Errors become a 500 with the message.
func Reply(result object.Object) (status int, headers map[string]string, body string) {
	headers = map[string]string{}
	if r, ok := result.(*Response); ok {
		status, headers = r.Status, r.Headers
		result = r.Body
	}
	if err, ok := result.(*object.Error); ok {
		return 500, map[string]string{"Content-Type": "text/plain; charset=utf-8"}, err.Message + "\n"
	}

	switch result := result.(type) {
	case *object.Null:
		if status == 0 {
			status = 204
		}
		return status, headers, ""
	case *object.String:
		body = result.Value
	default:
		var out strings.Builder
		if err := encodeJSON(&out, result, "", 0); err != nil {
			return 500, map[string]string{"Content-Type": "text/plain; charset=utf-8"}, err.Message + "\n"
		}
		body = out.String()
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = "application/json"
		}
	}
	if status == 0 {
		status = 200
	}
	return status, headers, body
}

func init() {
	builtins["serve"] = &object.Builtin{Name: "serve", Fn: serveBuiltin}
	builtins["response"] = &object.Builtin{Name: "response", Fn: responseBuiltin}
}

func serveBuiltin(args ...object.Object) object.Object {
	if len(args) != 2 {
		return wrongArgs(0, 0, "serve", len(args), "2")
	}
	port, ok := args[0].(*object.Integer)
	if !ok || port.Value < 0 || port.Value > 65535 {
		return &object.Error{Message: "the port for serve must be an integer from 0 to 65535, got " + args[0].Inspect()}
	}
	handler := args[1]
	if !isCallable(handler) {
		return wrongType(0, 0, "second argument to `serve`", "a function", handler)
	}
	if server == nil {
		return noSystem("serve")
	}

	err := server.Serve(int(port.Value), func(request *object.Hash) object.Object {
		return applyFunction(handler, []object.Object{request})
	})
	if err != nil {
		return &object.Error{Message: "serve: " + err.Error()}
	}
	return NULL
}

// responseBuiltin is response(body) or response(body, {status, headers}).
func responseBuiltin(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return wrongArgs(0, 0, "response", len(args), "1 or 2")
	}
	r := &Response{Status: 200, Headers: map[string]string{}, Body: args[0]}
	if len(args) == 1 {
		return r
	}

	options, ok := args[1].(*object.Hash)
	if !ok {
		return wrongType(0, 0, "the options of response", "an object", args[1])
	}
	for _, pair := range options.OrderedPairs() {
		name := objectToString(pair.Key)
		switch name {
		case "status":
			status, ok := pair.Value.(*object.Integer)
			if !ok || status.Value < 100 || status.Value > 999 {
				return &object.Error{Message: "the status of a response must be an integer from 100 to 999, got " + pair.Value.Inspect()}
			}
			r.Status = int(status.Value)
		case "headers":
			headers, ok := pair.Value.(*object.Hash)
			if !ok {
				return wrongType(0, 0, "the headers of a response", "an object", pair.Value)
			}
			for _, header := range headers.OrderedPairs() {
				r.Headers[objectToString(header.Key)] = objectToString(header.Value)
			}
		default:
			return &object.Error{Message: strfmt.Sprintf("response has no option %q, only status and headers", name)}
		}
	}
	return r
}

package evaluator

import (
	"strings"

	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/order"
	"github.com/migueltarga/TargaScript/strfmt"
)

func init() {
	builtins["fetch"] = &object.Builtin{Name: "fetch", Fn: fetch}
}

// fetch makes an HTTP request and returns a promise of the response:
//
//	let response = await fetch(url, {"method": "POST", "body": {"a": 1}})
//
// Options are method, headers (an object) and body. A body that isn't a
// string is sent as JSON. The response has status, ok, url, headers, text
// (the body) and json(), which parses the body.
func fetch(args ...object.Object) object.Object {
	return fetchWithHost(host, args...)
}

func fetchWithHost(h Host, args ...object.Object) object.Object {
	if len(args) != 1 && len(args) != 2 {
		return wrongArgs(0, 0, "fetch", len(args), "1 or 2")
	}
	url, ok := args[0].(*object.String)
	if !ok {
		return wrongType(0, 0, "first argument to fetch", "a URL string", args[0])
	}

	request := FetchRequest{URL: url.Value, Method: "GET", Headers: map[string]string{}}
	if len(args) == 2 {
		options, ok := args[1].(*object.Hash)
		if !ok {
			return wrongType(0, 0, "options for fetch", "an object", args[1])
		}
		if err := readFetchOptions(options, &request); err != nil {
			return err
		}
	}

	if h == nil {
		return &object.Error{Message: "fetch is not available here"}
	}

	promise := object.NewPromise()
	h.Fetch(request, func(response FetchResponse, err error) {
		if err != nil {
			promise.Reject(&object.Error{Message: strfmt.Sprintf("fetch %s failed: %s", request.URL, err)})
			return
		}
		promise.Resolve(responseObject(response))
	})
	return promise
}

func readFetchOptions(options *object.Hash, request *FetchRequest) *object.Error {
	for _, pair := range options.OrderedPairs() {
		key := objectToString(pair.Key)
		switch key {
		case "method":
			method, ok := pair.Value.(*object.String)
			if !ok {
				return &object.Error{Message: "fetch option method must be a string"}
			}
			request.Method = strings.ToUpper(method.Value)
		case "headers":
			headers, ok := pair.Value.(*object.Hash)
			if !ok {
				return &object.Error{Message: "fetch option headers must be an object"}
			}
			for _, header := range headers.OrderedPairs() {
				request.Headers[objectToString(header.Key)] = objectToString(header.Value)
			}
		case "body":
			if body, ok := pair.Value.(*object.String); ok {
				request.Body = body.Value
				continue
			}
			encoded := toJSON(pair.Value)
			if err, ok := encoded.(*object.Error); ok {
				return err
			}
			request.Body = encoded.(*object.String).Value
			if !hasHeader(request.Headers, "Content-Type") {
				request.Headers["Content-Type"] = "application/json"
			}
		default:
			return &object.Error{Message: strfmt.Sprintf("unknown fetch option %q, expected method, headers or body", key)}
		}
	}
	return nil
}

func hasHeader(headers map[string]string, name string) bool {
	for key := range headers {
		if strings.EqualFold(key, name) {
			return true
		}
	}
	return false
}

// responseObject is what a fetch promise resolves to.
func responseObject(response FetchResponse) *object.Hash {
	headers := object.NewHash()
	names := make([]string, 0, len(response.Headers))
	for name := range response.Headers {
		names = append(names, name)
	}
	order.Strings(names)
	for _, name := range names {
		headers.Set(&object.String{Value: strings.ToLower(name)}, &object.String{Value: response.Headers[name]})
	}

	body := &object.String{Value: response.Body}
	hash := object.NewHash()
	hash.Set(&object.String{Value: "status"}, &object.Integer{Value: int64(response.Status)})
	hash.Set(&object.String{Value: "ok"}, nativeBoolToBooleanObject(response.Status >= 200 && response.Status < 300))
	hash.Set(&object.String{Value: "url"}, &object.String{Value: response.URL})
	hash.Set(&object.String{Value: "headers"}, headers)
	hash.Set(&object.String{Value: "text"}, body)
	hash.Set(&object.String{Value: "json"}, &object.Builtin{Name: "json", Fn: func(args ...object.Object) object.Object {
		return parseJSON(body)
	}})
	return hash
}

package eventloop

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/migueltarga/TargaScript/evaluator"
)

var client = &http.Client{Timeout: 60 * time.Second}

// Fetch makes an HTTP request in another goroutine and calls done with the
// response on the loop.
func (l *Loop) Fetch(request evaluator.FetchRequest, done func(evaluator.FetchResponse, error)) {
	l.Go(func() func() {
		response, err := doFetch(request)
		return func() { done(response, err) }
	})
}

func doFetch(request evaluator.FetchRequest) (evaluator.FetchResponse, error) {
	var body io.Reader
	if request.Body != "" {
		body = strings.NewReader(request.Body)
	}
	req, err := http.NewRequest(request.Method, request.URL, body)
	if err != nil {
		return evaluator.FetchResponse{}, err
	}
	for name, value := range request.Headers {
		req.Header.Set(name, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return evaluator.FetchResponse{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return evaluator.FetchResponse{}, err
	}

	headers := map[string]string{}
	for name, values := range resp.Header {
		headers[name] = strings.Join(values, ", ")
	}
	return evaluator.FetchResponse{
		URL:     resp.Request.URL.String(),
		Status:  resp.StatusCode,
		Headers: headers,
		Body:    string(data),
	}, nil
}

package eventloop

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/migueltarga/TargaScript/evaluator"
)

var client = &http.Client{Timeout: 60 * time.Second}

// Fetch makes an HTTP request in another goroutine and calls done with the
// response on the loop.
func (l *Loop) Fetch(request evaluator.FetchRequest, done func(evaluator.FetchResponse, error)) {
	l.Go(func() func() {
		response, err := l.doFetch(request)
		return func() { done(response, err) }
	})
}

func (l *Loop) checkFetchURL(raw string) error {
	if !l.restricted {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return fmt.Errorf("fetch requires an HTTP or HTTPS URL")
	}
	if !l.allowedHosts[strings.ToLower(u.Hostname())] {
		return fmt.Errorf("fetch to %q is not allowed", u.Hostname())
	}
	return nil
}

func (l *Loop) doFetch(request evaluator.FetchRequest) (evaluator.FetchResponse, error) {
	if err := l.checkFetchURL(request.URL); err != nil {
		return evaluator.FetchResponse{}, err
	}
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

	requestClient := client
	if l.restricted {
		copy := *client
		copy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return l.checkFetchURL(req.URL.String())
		}
		requestClient = &copy
	}
	resp, err := requestClient.Do(req)
	if err != nil {
		return evaluator.FetchResponse{}, err
	}
	defer resp.Body.Close()

	reader := io.Reader(resp.Body)
	if l.restricted {
		reader = io.LimitReader(reader, l.maxFetchBytes+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return evaluator.FetchResponse{}, err
	}
	if l.restricted && int64(len(data)) > l.maxFetchBytes {
		return evaluator.FetchResponse{}, fmt.Errorf("fetch response exceeds %d bytes", l.maxFetchBytes)
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

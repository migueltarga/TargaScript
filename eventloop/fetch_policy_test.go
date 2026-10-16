package eventloop

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/evaluator"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestRestrictedFetchHosts(t *testing.T) {
	l := NewRestricted()
	for _, url := range []string{"https://api.example.com/data", "http://127.0.0.1/"} {
		if l.checkFetchURL(url) == nil {
			t.Errorf("allowed %s before opt-in", url)
		}
	}
	l.AllowFetchHosts("API.Example.com")
	for _, url := range []string{"https://api.example.com/data", "http://api.example.com:8080/data"} {
		if err := l.checkFetchURL(url); err != nil {
			t.Errorf("rejected allowed host %s: %v", url, err)
		}
	}
	for _, url := range []string{"https://other.example.com/data", "file:///tmp/a", "https://api.example.com.evil.test/"} {
		if l.checkFetchURL(url) == nil {
			t.Errorf("allowed unexpected URL %s", url)
		}
	}
}

func TestRestrictedFetchLimitsResponseAndRedirects(t *testing.T) {
	previous := client
	client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/redirect" {
			return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {"https://other.example.com/"}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("four")), Request: req}, nil
	})}
	t.Cleanup(func() { client = previous })

	l := NewRestricted()
	l.AllowFetchHosts("api.example.com")
	l.SetMaxFetchBytes(3)
	if _, err := l.doFetch(evaluator.FetchRequest{URL: "https://api.example.com/data"}); err == nil || !strings.Contains(err.Error(), "exceeds 3 bytes") {
		t.Errorf("response limit: %v", err)
	}
	if _, err := l.doFetch(evaluator.FetchRequest{URL: "https://api.example.com/redirect"}); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("redirect policy: %v", err)
	}
}

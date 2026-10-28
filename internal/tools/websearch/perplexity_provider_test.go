package websearch

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const perplexityTestBody = `{"choices":[{"message":{"content":"ok"}}],"citations":["https://example.com"]}`

type captureTransport struct {
	req *http.Request
}

func (c *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.req = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(perplexityTestBody)),
		Request:    req,
	}, nil
}

func TestPerplexityDriverSendsIntegrationHeaderToPublicAPI(t *testing.T) {
	capture := &captureTransport{}
	orig := http.DefaultTransport
	http.DefaultTransport = capture
	t.Cleanup(func() { http.DefaultTransport = orig })

	d := &perplexityDriver{}
	if _, err := d.Search(context.Background(), SearchRequest{Query: "q"}, ProviderConfig{APIKey: "pplx-key"}); err != nil {
		t.Fatalf("search returned error: %v", err)
	}
	if capture.req == nil || capture.req.URL.Host != "api.perplexity.ai" {
		t.Fatalf("expected request to api.perplexity.ai, got %#v", capture.req)
	}
	if got := capture.req.Header.Values("X-Pplx-Integration"); len(got) != 1 || got[0] != "goclaw" {
		t.Fatalf("expected single X-Pplx-Integration goclaw, got %#v", got)
	}
}

func TestPerplexityDriverOmitsIntegrationHeaderForCustomBaseURL(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Values("X-Pplx-Integration")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(perplexityTestBody))
	}))
	defer srv.Close()

	d := &perplexityDriver{}
	if _, err := d.Search(context.Background(), SearchRequest{Query: "q"}, ProviderConfig{APIKey: "pplx-key", BaseURL: srv.URL}); err != nil {
		t.Fatalf("search returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no X-Pplx-Integration for custom base URL, got %#v", got)
	}
}

package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCleanupPrompt_notEmpty(t *testing.T) {
	if strings.TrimSpace(CleanupPrompt) == "" {
		t.Error("CleanupPrompt must not be empty")
	}
}

func TestCleanupPrompt_containsKeyGuidance(t *testing.T) {
	// Verify the prompt preserves technical terms as intended.
	if !strings.Contains(CleanupPrompt, "technical terms") {
		t.Error("CleanupPrompt should mention technical terms")
	}
	if !strings.Contains(CleanupPrompt, "filler words") {
		t.Error("CleanupPrompt should mention filler words")
	}
}

// fakeServer builds a test HTTP server that returns the given response body
// with the given status code.
func fakeServer(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// successBody returns a minimal Claude API response JSON for the given text.
func successBody(text string) string {
	resp := map[string]any{
		"content": []map[string]string{
			{"type": "text", "text": text},
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func TestProcess_success(t *testing.T) {
	srv := fakeServer(http.StatusOK, successBody("Hello world."))
	defer srv.Close()

	c := New("test-key", "claude-test", 5)
	c.http = srv.Client()
	// Override the endpoint via a patched request — use a round-tripper that rewrites URL.
	c.http = &http.Client{
		Transport: &rewriteTransport{base: http.DefaultTransport, target: srv.URL},
	}

	got, err := c.Process("system", "hello world um")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Hello world." {
		t.Errorf("expected 'Hello world.', got %q", got)
	}
}

func TestProcess_apiError(t *testing.T) {
	srv := fakeServer(http.StatusUnauthorized, `{"error":"invalid api key"}`)
	defer srv.Close()

	c := New("bad-key", "claude-test", 5)
	c.http = &http.Client{
		Transport: &rewriteTransport{base: http.DefaultTransport, target: srv.URL},
	}

	_, err := c.Process("system", "text")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got: %v", err)
	}
}

func TestProcess_emptyContent(t *testing.T) {
	body := `{"content":[]}`
	srv := fakeServer(http.StatusOK, body)
	defer srv.Close()

	c := New("key", "model", 5)
	c.http = &http.Client{
		Transport: &rewriteTransport{base: http.DefaultTransport, target: srv.URL},
	}

	_, err := c.Process("system", "text")
	if err == nil {
		t.Fatal("expected error for empty content array")
	}
}

func TestProcess_malformedJSON(t *testing.T) {
	srv := fakeServer(http.StatusOK, `{bad json`)
	defer srv.Close()

	c := New("key", "model", 5)
	c.http = &http.Client{
		Transport: &rewriteTransport{base: http.DefaultTransport, target: srv.URL},
	}

	_, err := c.Process("system", "text")
	if err == nil {
		t.Fatal("expected error for malformed JSON response")
	}
}

// rewriteTransport redirects all requests to target (the test server URL).
type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (r *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = strings.TrimPrefix(r.target, "http://")
	return r.base.RoundTrip(req2)
}

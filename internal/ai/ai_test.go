package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGemini_NoKey(t *testing.T) {
	g := NewGemini("", "")
	_, err := g.Generate(context.Background(), "diff")
	if !errors.Is(err, ErrNoAPIKey) {
		t.Errorf("want ErrNoAPIKey, got %v", err)
	}
}

func TestClaude_NoKey(t *testing.T) {
	c := NewClaude("", "")
	_, err := c.Generate(context.Background(), "diff")
	if !errors.Is(err, ErrNoAPIKey) {
		t.Errorf("want ErrNoAPIKey, got %v", err)
	}
}

func TestOpenAI_NoKey(t *testing.T) {
	o := NewOpenAI("", "")
	_, err := o.Generate(context.Background(), "diff")
	if !errors.Is(err, ErrNoAPIKey) {
		t.Errorf("want ErrNoAPIKey, got %v", err)
	}
}

func TestGemini_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "k" {
			t.Errorf("missing api key header")
		}
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"feat: add x"}]},"finishReason":"STOP"}]}`))
	}))
	defer srv.Close()
	prev := HTTPClient
	HTTPClient = srv.Client()
	defer func() { HTTPClient = prev }()

	g := NewGemini("k", "model-x")
	g.BaseURL = srv.URL

	got, err := g.Generate(context.Background(), "diff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "feat: add x" {
		t.Errorf("got %q", got)
	}
}

func TestGemini_TruncatedUsesFirstLine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"feat: add x\n\n- detail"}]},"finishReason":"MAX_TOKENS"}]}`))
	}))
	defer srv.Close()
	prev := HTTPClient
	HTTPClient = srv.Client()
	defer func() { HTTPClient = prev }()

	g := NewGemini("k", "")
	g.BaseURL = srv.URL

	got, err := g.Generate(context.Background(), "diff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "feat: add x" {
		t.Errorf("got %q, want first line", got)
	}
}

func TestClaude_APIErrorPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
	}))
	defer srv.Close()
	prev := HTTPClient
	HTTPClient = srv.Client()
	defer func() { HTTPClient = prev }()

	c := NewClaude("k", "")
	c.BaseURL = srv.URL

	_, err := c.Generate(context.Background(), "diff")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected 'boom' error, got %v", err)
	}
}

func TestOpenAI_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer k" {
			t.Errorf("auth header = %q", got)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"chore: bump deps"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()
	prev := HTTPClient
	HTTPClient = srv.Client()
	defer func() { HTTPClient = prev }()

	o := NewOpenAI("k", "gpt-test")
	o.BaseURL = srv.URL
	got, err := o.Generate(context.Background(), "diff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "chore: bump deps" {
		t.Errorf("got %q", got)
	}
}

func TestProvidersImplementInterface(t *testing.T) {
	var _ Provider = NewGemini("", "")
	var _ Provider = NewClaude("", "")
	var _ Provider = NewOpenAI("", "")
}

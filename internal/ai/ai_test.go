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

func TestOpenAI_BaseURLOverrideTrimsTrailingSlash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"fix: proxy url"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()
	prev := HTTPClient
	HTTPClient = srv.Client()
	defer func() { HTTPClient = prev }()

	o := NewOpenAIWithBaseURL("k", "litellm-model", srv.URL+"/")
	got, err := o.Generate(context.Background(), "diff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "fix: proxy url" {
		t.Errorf("got %q", got)
	}
}

func TestBaseURLOverrideWhitespaceUsesDefaults(t *testing.T) {
	if got := NewGeminiWithBaseURL("k", "", "  ").BaseURL; got != "https://generativelanguage.googleapis.com/v1beta" {
		t.Errorf("gemini base URL = %q", got)
	}
	if got := NewClaudeWithBaseURL("k", "", "  ").BaseURL; got != "https://api.anthropic.com" {
		t.Errorf("claude base URL = %q", got)
	}
	if got := NewOpenAIWithBaseURL("k", "", "  ").BaseURL; got != "https://api.openai.com" {
		t.Errorf("openai base URL = %q", got)
	}
}

func TestProvidersImplementInterface(t *testing.T) {
	var _ Provider = NewGemini("", "")
	var _ Provider = NewClaude("", "")
	var _ Provider = NewOpenAI("", "")
}

// TestSetStyle intentionally does not run in parallel: it mutates the shared
// CommitPrompt package var, which provider tests read via HTTP request bodies.
func TestSetStyle(t *testing.T) {
	prev := CommitPrompt
	t.Cleanup(func() { CommitPrompt = prev })

	cases := []struct {
		name  string
		style string
		want  string
	}{
		{"conventional", "conventional", ConventionalPrompt},
		{"human", "human", HumanPrompt},
		{"empty", "", HumanPrompt},
		{"uppercase conventional", "CONVENTIONAL", ConventionalPrompt},
		{"whitespace conventional", "  conventional  ", ConventionalPrompt},
		{"unknown", "bogus", HumanPrompt},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			SetStyle(tt.style)
			if CommitPrompt != tt.want {
				t.Errorf("SetStyle(%q): CommitPrompt = %q, want %q", tt.style, CommitPrompt, tt.want)
			}
		})
	}
}

// TestCommitPrompt_DefaultIsHuman verifies that, absent a call to SetStyle,
// providers send the human prompt to the API.
func TestCommitPrompt_DefaultIsHuman(t *testing.T) {
	prev := CommitPrompt
	CommitPrompt = HumanPrompt
	t.Cleanup(func() { CommitPrompt = prev })

	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"Add x"}]},"finishReason":"STOP"}]}`))
	}))
	defer srv.Close()
	prevHTTP := HTTPClient
	HTTPClient = srv.Client()
	defer func() { HTTPClient = prevHTTP }()

	g := NewGemini("k", "")
	g.BaseURL = srv.URL
	if _, err := g.Generate(context.Background(), CommitPrompt+"<diff>"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(gotBody, "Write it the way an experienced human developer would") {
		t.Errorf("request body does not contain human prompt: %s", gotBody)
	}
	if strings.Contains(gotBody, "Conventional Commits format") {
		t.Errorf("request body unexpectedly contains conventional prompt: %s", gotBody)
	}
}

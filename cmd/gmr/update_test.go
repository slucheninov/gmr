package main

import (
	"errors"
	"github.com/slucheninov/gmr/internal/version"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseUpdateArgs(t *testing.T) {
	t.Parallel()
	for _, flag := range []string{"-u", "--update"} {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			opts, err := parseGmrArgs([]string{flag})
			if err != nil || opts != (gmrOptions{update: true}) {
				t.Fatalf("options = %+v, error = %v", opts, err)
			}
			for _, other := range []string{"-m", "--message", "-c", "--commit", "-s", "--stay", "feature"} {
				for _, args := range [][]string{{flag, other}, {other, flag}} {
					if _, err := parseGmrArgs(args); err == nil || !strings.Contains(err.Error(), "-u must be used alone") {
						t.Errorf("parseGmrArgs(%v) error = %v", args, err)
					}
				}
			}
			if _, err := parseGmrArgs([]string{flag, "--help"}); !errors.Is(err, errShowHelp) {
				t.Fatalf("--help error = %v", err)
			}
		})
	}
}

type updateRoundTripper func(*http.Request) (*http.Response, error)

func (f updateRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRunUpdateOutsideRepository(t *testing.T) {
	// No git executable, repository, or real network is needed to check updates.
	t.Chdir(t.TempDir())
	t.Setenv("PATH", "")
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	requests := 0
	http.DefaultTransport = updateRoundTripper(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.Method != http.MethodGet || r.URL.String() != "https://api.github.com/repos/slucheninov/gmr/releases/latest" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v` + strings.TrimPrefix(version.Version, "v") + `"}`)),
			Header:     make(http.Header),
		}, nil
	})
	if err := run(gmrOptions{update: true}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

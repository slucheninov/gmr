// Package ai contains AI providers that turn a git diff into a commit message.
package ai

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// CommitPrompt is the instruction prepended to every diff sent to a provider.
const CommitPrompt = `Generate a git commit message for this diff using Conventional Commits format.
Rules:
- Format: scope: description
- Scope is optional, use only if obvious from the diff
- Description: imperative mood, lowercase, no period, max 72 chars
- If changes are significant, add a body after a blank line (max 3 bullet points)
- Reply ONLY with the commit message, no markdown, no explanation

`

// ErrNoAPIKey is returned by a Provider when its required API key is missing.
var ErrNoAPIKey = errors.New("no API key")

// Provider generates a commit message from a unified diff.
type Provider interface {
	Name() string
	Generate(ctx context.Context, diff string) (string, error)
}

// HTTPClient is the http.Client all providers share. Tests override it.
var HTTPClient = &http.Client{Timeout: 30 * time.Second}

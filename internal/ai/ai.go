// Package ai contains AI providers that turn a prompt (a commit diff or a
// release log) into generated text.
package ai

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

// HumanPrompt asks for a plain-English commit message without Conventional Commits tags.
const HumanPrompt = `Generate a git commit message for this diff. Write it the way an experienced human developer would.
Rules:
- Title: a plain English sentence in imperative mood, starting with a capital letter, no trailing period, max 72 chars
- NEVER prefix the title with a Conventional Commits type ("feat:", "fix:", "chore:", "refactor:", "docs:", ...) or any other "word:" tag
- NEVER use parentheses for scope in the title
- Describe what the change does and why it matters, not which files were touched
- If the change is significant, add a body after a blank line (max 3 bullet points, plain sentences)
- Reply ONLY with the commit message, no markdown, no explanation

`

// ConventionalPrompt asks for a Conventional Commits message. Opt in via GMR_COMMIT_STYLE=conventional.
const ConventionalPrompt = `Generate a git commit message for this diff using Conventional Commits format.
Rules:
- Format: "type: description" or "scope: description"
- NEVER use parentheses in the title (no "feat(x):" or "fix(y):")
- Description: imperative mood, lowercase, no period, max 72 chars
- If changes are significant, add a body after a blank line (max 3 bullet points)
- Reply ONLY with the commit message, no markdown, no explanation

`

// CommitPrompt is the instruction prepended to every diff sent to a provider.
// Providers read it at call time; SetStyle swaps it.
var CommitPrompt = HumanPrompt

// SetStyle selects the commit-message style: "conventional" for Conventional
// Commits, anything else (including "" and "human") for plain human style.
func SetStyle(style string) {
	if strings.EqualFold(strings.TrimSpace(style), "conventional") {
		CommitPrompt = ConventionalPrompt
		return
	}
	CommitPrompt = HumanPrompt
}

// ReleasePrompt asks a provider to pick a semver bump and write release notes
// from a git log. The reply must start with a "BUMP: <patch|minor|major>" line,
// then a "---" separator, then the notes.
const ReleasePrompt = `You are writing a release for a software project. Below is the git log since the previous release.

Decide the semantic version bump and write the release notes.
Rules:
- First line must be exactly "BUMP: patch", "BUMP: minor" or "BUMP: major"
  - major: breaking changes to the public interface / CLI flags / config
  - minor: new user-visible features or new options
  - patch: bug fixes, docs, refactors, chores only
- Second line must be exactly "---"
- Then the release notes in Markdown: a one-sentence summary, then short bullet points grouped under "### Added", "### Changed", "### Fixed" (omit empty groups)
- Write for humans reading a changelog: say what changed for the user, do not restate commit hashes or file names
- Do NOT invent changes that are not in the log
- Reply ONLY with that block, no code fences, no explanation

`

// ErrNoAPIKey is returned by a Provider when its required API key is missing.
var ErrNoAPIKey = errors.New("no API key")

// Provider turns a prompt into generated text.
type Provider interface {
	Name() string
	Generate(ctx context.Context, prompt string) (string, error)
}

// HTTPClient is the http.Client all providers share. Tests override it.
var HTTPClient = &http.Client{Timeout: 30 * time.Second}

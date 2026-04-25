# Contributing to gmr

First off — thanks for taking the time to contribute! 🎉

This document describes the workflow for proposing changes to `gmr`. By
participating in this project, you agree to abide by the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Ways to contribute

- **Report a bug** — open an [issue](https://github.com/slucheninov/gmr/issues/new)
  with steps to reproduce, expected behavior, and your environment
  (`gmr --version`, OS, `go version` if relevant).
- **Suggest a feature** — open an issue describing the use case *before* you
  start coding. Non-trivial changes without prior discussion may be rejected.
- **Improve docs** — typo fixes, clarifications, and new examples are always
  welcome.
- **Submit a pull request** — bug fixes, new AI providers, new platform
  integrations, test coverage improvements.
- **Help with security** — see [SECURITY.md](SECURITY.md) for the responsible
  disclosure process.

## Development setup

### Prerequisites

- Go **1.25+**
- `git`
- Optional: [`golangci-lint`](https://golangci-lint.run/) v2 (matches CI)
- Optional, for end-to-end testing: `gh` and/or `glab`, plus an AI API key

### Clone and build

```bash
git clone https://github.com/slucheninov/gmr.git
cd gmr
go build ./cmd/gmr
./gmr --version
```

### Run tests

```bash
go test -race ./...
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### Lint

```bash
go vet ./...
golangci-lint run
```

CI runs the same commands on Go 1.25 — green locally → green in CI.

## Pull request workflow

1. **Fork** the repo and create your branch from `master`:
   ```bash
   git checkout -b feat/my-thing
   ```
2. **Implement** your change. Keep PRs focused — one logical change per PR.
3. **Add tests** for new behavior:
   - Pure logic → standard `*_test.go` next to the file.
   - AI providers → `httptest`-backed tests that override `ai.HTTPClient`.
   - `git`-touching code → use the `git.Runner` interface and inject a fake.
4. **Update the changelog** under the `## [Unreleased]` section in
   [`CHANGELOG.md`](CHANGELOG.md). Use the categories `Added` / `Changed` /
   `Fixed` / `Removed`.
5. **Update docs** if you changed user-visible behavior (flags, env vars,
   workflow). README and `internal/<pkg>/doc` comments should stay in sync.
6. **Run the full check suite** locally:
   ```bash
   go vet ./... && golangci-lint run && go test -race ./...
   ```
7. **Open a PR** against `master`. Fill in the description with:
   - What problem this solves and why.
   - Screenshots or terminal output for UX changes.
   - Linked issue(s), if any (`Closes #123`).

CI must be green before merge. A maintainer will review and may request
changes — please don't take review feedback personally, it is about the code,
not about you.

## Coding guidelines

- **Formatting** — `gofmt` / `goimports`. Run `go fmt ./...` before committing.
- **Style** — follow [Effective Go](https://go.dev/doc/effective_go) and
  the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
- **Errors** — return them; do not log-and-swallow. The CLI layer
  (`cmd/gmr/main.go`) is the only place that calls `ui.Errf`.
- **Side effects** — keep `internal/*` packages pure where possible. Put
  process-wide state (signal handling, stdin/stdout, `os.Exit`) in
  `cmd/gmr/main.go`.
- **Public API** — anything outside `internal/` is part of the importable
  Go API. Don't expose helpers there unless intentional.
- **Comments** — explain *why*, not *what*. Avoid narrating the obvious.
- **Commit messages** — Conventional Commits, e.g.:
  ```
  feat: add Mistral provider
  fix(git): handle detached HEAD on origin
  docs: clarify GMR_MAX_DIFF behavior
  ```

## Adding a new AI provider

The `ai.Provider` interface is the extension point:

```go
type Provider interface {
    Name() string
    Generate(ctx context.Context, diff string) (string, error)
}
```

Reference implementation: [`internal/ai/gemini.go`](internal/ai/gemini.go).

Checklist for a new provider:

1. New file `internal/ai/<name>.go` implementing `Provider`.
2. Honor `ai.HTTPClient` for HTTP calls (so tests can inject `httptest`).
3. Use `ai.CommitPrompt` as the system prompt prefix.
4. Return `ai.ErrNoAPIKey` when the key is empty.
5. Handle truncation (`finishReason: MAX_TOKENS` etc.) by falling back to the
   first line of the response.
6. Wire it into the chain in `cmd/gmr/main.go` (preserve order:
   Gemini → Claude → OpenAI → new ones go to the end unless replacing).
7. Add tests in `internal/ai/ai_test.go` covering: missing key, success, API
   error payload, truncated response.
8. Document the env var in [README.md](README.md) → Configuration and update
   [CHANGELOG.md](CHANGELOG.md).

## Releasing

Releases are cut by maintainers via a tag push:

1. Bump `Version` in `internal/version/version.go`.
2. Move `[Unreleased]` entries in `CHANGELOG.md` under a dated
   `[X.Y.Z] - YYYY-MM-DD` heading.
3. Commit, tag `vX.Y.Z`, push the tag — `.github/workflows/release.yml` does
   the rest (build matrix, archives, checksums, GitHub Release).

## Questions?

Open a [Discussion](https://github.com/slucheninov/gmr/discussions) or an
issue. We'll do our best to help.

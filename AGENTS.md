# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## Overview

`gmr` (Git Merge Request) is a Go CLI that automates the merge request / pull request workflow. It stages changes, generates a commit message via AI (Gemini → Claude → OpenAI → manual), creates a branch, commits, and opens a GitLab MR or GitHub PR — all in one command. Platform is auto-detected from the `origin` remote URL.

## Layout

```
cmd/gmr/main.go             # CLI entry point and orchestration
internal/ai/                # Gemini / Claude / OpenAI providers (Provider interface)
internal/git/               # git wrapper (Runner interface — testable)
internal/platform/          # platform detection + GitLab project path parsing
internal/commit/            # commit-message helpers (title, body, MR description)
internal/ui/                # logging + ANSI colors (honors NO_COLOR)
internal/version/           # Version constant (override via -ldflags)
```

## Usage

```bash
gmr [options] [branch-name]   # branch-name defaults to auto/YYYYMMDD-HHMMSS
gmr -m              # generate commit message only (prints to stdout)
gmr -s              # after MR/PR, stay on the feature branch (no checkout to main)
gmr -h | -v
```

## Build / Test / Lint

```bash
go build ./cmd/gmr
go test -race ./...
go vet ./...
```

## Dependencies

- Go 1.25+
- `glab` (GitLab CLI) or `gh` (GitHub CLI) — only at runtime, not for building
- `git`
- `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, and/or `OPENAI_API_KEY` (at least one required)

## Configuration (env vars)

- `GMR_MAIN_BRANCH` — base branch (default: auto-detected from `origin/HEAD`, fallback: `main`/`master`)
- `GMR_GEMINI_MODEL` — Gemini model (default: `gemini-flash-latest`)
- `GMR_ANTHROPIC_MODEL` — Anthropic model (default: `claude-sonnet-4-20250514`)
- `GMR_OPENAI_MODEL` — OpenAI model (default: `gpt-4o-mini`)
- `GMR_MAX_DIFF` — max diff lines sent to API (default: `500`)
- `EDITOR` — editor for the `e(edit)` choice (default: `vim`)
- `NO_COLOR` — disable ANSI colors

## Rules for changes

- **Version**: bump `Version` in `internal/version/version.go` (semver: patch for fixes, minor for features, major for breaking).
- **Changelog**: always update `CHANGELOG.md` (Added/Changed/Fixed/Removed under a new version section).
- **Tests**: extend tests in `internal/<pkg>/*_test.go` for new behavior; AI providers must use `httptest` and override `ai.HTTPClient`.
- **README**: update `README.md` if changes affect user-facing info (new flags, new env vars, install instructions, workflow).
- **Releases**: cut by tagging `vX.Y.Z` and pushing — `.github/workflows/release.yml` builds `linux/{amd64,arm64}` + `darwin/{amd64,arm64}` tarballs and a GitHub Release.

## Notes

- UI messages are in Ukrainian / English mixed (mirrors the original tool).
- `ui.Log/OK/Warn/Errf` write to `stderr`. `gmr -m` writes the commit message to `stdout` so the output is pipe-friendly.
- `ai.Provider` is the extension point for new providers; keep them stateless and inject `HTTPClient` via the package var so tests can swap it.

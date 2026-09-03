# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Overview

`gmr` (Git Merge Request) is a Go CLI that automates the merge request / pull request workflow. It stages changes, generates a commit message via AI (Gemini → Claude → OpenAI → manual), creates a branch, commits, and opens a GitLab MR or GitHub PR — all in one command. It also cuts release tags (`gmr deploy`) and reports CI status (`gmr status`). Platform is auto-detected from the `origin` remote URL.

## Layout

```
cmd/gmr/main.go             # CLI entry point, subcommand dispatch, MR/PR flow
cmd/gmr/deploy.go           # `gmr deploy` — next semver tag + release notes
cmd/gmr/status.go           # `gmr status` — CI pipeline status + verdict
internal/ai/                # Gemini / Claude / OpenAI providers (Provider interface)
internal/git/               # git wrapper (Runner interface — testable)
internal/platform/          # platform detection + GitLab project path parsing
internal/commit/            # commit-message helpers (title, body, MR description)
internal/release/           # semver parsing, next-tag calculation, AI reply parsing
internal/ci/                # gh / glab pipeline status (Runner interface — testable)
internal/ui/                # logging + ANSI colors (honors NO_COLOR)
internal/version/           # Version constant (override via -ldflags)
```

## Usage

```bash
gmr [options] [branch-name]   # branch-name defaults to a name derived from the commit title
gmr -m              # generate commit message only (prints to stdout)
gmr -s              # after MR/PR, stay on the feature branch (skips the stay/switch question)
gmr deploy [tag]    # tag the next release; --patch/--minor/--major, --no-release, -y
gmr status [ref]    # CI status for current branch + latest tag; --limit N; exit 1 on failure
gmr -h | -v
```

`deploy` and `status` are reserved as the first argument.

## Build / Test / Lint

```bash
go build ./...
go test -race ./...
go vet ./...
gofmt -l .          # must print nothing
```

## Dependencies

- Go 1.25+
- `glab` (GitLab CLI) or `gh` (GitHub CLI) — only at runtime, not for building
- `git`
- `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, and/or `OPENAI_API_KEY` (at least one required)

## Configuration (env vars)

- `GEMINI_MODEL` / `ANTHROPIC_MODEL` / `OPENAI_MODEL` — model overrides (defaults: `gemini-flash-latest`, `claude-sonnet-4-20250514`, `gpt-4o-mini`)
- `GEMINI_BASE_URL` / `ANTHROPIC_BASE_URL` / `OPENAI_BASE_URL` — API base URL overrides (e.g. LiteLLM)
- `GMR_PROVIDERS` — AI provider fallback order (default: `gemini,claude,openai`)
- `GMR_COMMIT_STYLE` — `human` (default) or `conventional`
- `GMR_MAIN_BRANCH` — base branch (default: auto-detected from `origin/HEAD`)
- `GMR_MAX_DIFF` — max diff/log lines sent to the API (default: `500`)
- `GMR_TAG_PREFIX` — tag prefix for the first release when no tag exists (default: `v`)
- `EDITOR` — editor for the `e(edit)` choice (default: `vim`)
- `NO_COLOR` — disable ANSI colors

## Rules for changes

- **Version**: bump `Version` in `internal/version/version.go` (semver: patch for fixes, minor for features, major for breaking).
- **Changelog**: always update `CHANGELOG.md` (Added/Changed/Fixed/Removed under a new version section).
- **Tests**: extend tests in `internal/<pkg>/*_test.go` for new behavior. AI providers must use `httptest` and override `ai.HTTPClient`; anything shelling out must go through a `Runner` interface with a fake in tests.
- **README**: update `README.md` if changes affect user-facing info (new flags, env vars, install instructions, workflow).
- **Releases**: cut by tagging `vX.Y.Z` and pushing — `.github/workflows/release.yml` builds the tarballs and the GitHub Release. Because that workflow already creates the release, use `gmr deploy --no-release` in this repo.

## Notes

- UI messages are Ukrainian / English mixed (mirrors the original tool).
- `ui.Log/OK/Warn/Errf` write to `stderr`. `gmr -m` writes the commit message to `stdout` so the output is pipe-friendly.
- `ai.Provider` is the extension point for new providers; keep them stateless, take the full prompt as an argument, and inject `HTTPClient` via the package var so tests can swap it.

---

# Workflow conventions

## Model and delegation

The main session runs **Opus 5**. Opus writes are expensive, so plan and judge here, execute in Sonnet subagents.

| Operation | Where |
|---|---|
| Main chat, planning, review verdicts, synthesis | Opus 5 (here) |
| Any `Agent()` dispatch | `model: "sonnet"` |
| Heavy mechanical edits (rename, boilerplate) | `model: "haiku"` |

Dispatch when the work crosses these lines (self-enforced — there is no hook):

| Trigger | Action |
|---|---|
| Writing a file >30 lines | `Agent(general-purpose, sonnet)` |
| ≥2 edits, or any multi-file refactor | `Agent(general-purpose, sonnet)` |
| Reading ≥3 files for analysis | `Agent(Explore, sonnet)` |
| ≥3 shell commands for context | `Agent(general-purpose, sonnet)` |

If a threshold trips, the dispatch is the **first** tool call of the response — not after starting the work by hand. Pass exact paths, verbatim content, and acceptance criteria in the prompt; "I already have the context" is not a reason to skip. Batch independent dispatches into ONE message so they run in parallel.

Direct work stays small: 1-2 file reads, 1-2 orientation shell commands, one edit under ~10 lines.

```
Agent(subagent_type: "general-purpose", model: "sonnet",
      description: "<3-5 words>",
      prompt: "<paths, content/specs, acceptance criteria, lints to pass>")
```

## Skills

Invoke before acting, not after: `brainstorming` (before creative work) → `writing-plans` → `executing-plans` / `subagent-driven-development`. Also `systematic-debugging` (any bug), `test-driven-development`, `verification-before-completion` (before any "done" claim), `requesting-code-review`, `using-git-worktrees` (isolation), `dispatching-parallel-agents` (2+ independent tasks).

Workspace skills under `<repo>/.claude/skills/` win over global ones.

## Don't

- Don't disable the RTK hook (`~/.claude/hooks/rtk-rewrite.sh`) — it trims shell output tokens.
- Don't claim work is done without running the build/test/lint commands and reading the output.

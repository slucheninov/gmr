# AGENTS.md

Guidance for coding agents working in this repository.

## Agent instruction files

This file is the single source of truth. Edit only `AGENTS.md`; the other
files are thin pointers and must not duplicate its content.

| Agent | Entry point |
|---|---|
| ChatGPT / OpenAI Codex | `AGENTS.md` (read natively) |
| Grok Build (xAI) | `AGENTS.md` (read natively) |
| Claude Code | `CLAUDE.md` → imports `@AGENTS.md` |
| Gemini CLI | `GEMINI.md` → imports `@AGENTS.md` |

## Overview

`gmr` (Git Merge Request) is a Go CLI for GitLab and GitHub workflows. Its
default command stages changes, generates a commit message through the configured
AI provider chain (Gemini -> Claude -> OpenAI -> manual), creates or reuses a
feature branch, commits, pushes, and opens an MR/PR. It also provides `gmr deploy`
for releases and `gmr status` for CI/CD status. The platform is detected from the
`origin` remote URL.

## Project layout

```text
cmd/gmr/main.go             CLI entry point, argument parsing, and MR/PR flow
cmd/gmr/deploy.go           `gmr deploy` release orchestration
cmd/gmr/status.go           `gmr status` CI/CD reporting
cmd/gmr/update.go           `gmr -u` self-update orchestration
internal/ai/                AI provider interface and Gemini/Claude/OpenAI clients
internal/ci/                GitHub Actions and GitLab pipeline adapters
internal/commit/            commit title/body, branch name, and MR description helpers
internal/git/               git command wrapper and testable Runner interface
internal/platform/          platform detection and GitLab project-path parsing
internal/release/           semver, next-tag, and AI release-response helpers
internal/ui/                stderr logging and ANSI colors; honors NO_COLOR
internal/update/            release downloads, checksum verification, binary replacement
internal/version/           build version; overridable with -ldflags
```

Keep orchestration in `cmd/gmr` and reusable/testable behavior in the relevant
`internal` package.

## Commands

```bash
gmr [options] [branch-name] # commit changes and open an MR/PR
gmr -m                     # generate a commit message, then optionally commit it
gmr -c                     # commit and push to the current branch, no MR/PR
gmr -s                     # create MR/PR and stay on the feature branch
gmr deploy [options] [tag] # create and push a release tag
gmr status [options] [ref] # report recent CI/CD runs
gmr -u                     # update this executable to the latest stable release
gmr -h | -v
```

- An omitted branch name is derived from the generated commit title, with
  `auto-YYYYMMDD-HHMMSS` as the final fallback.
- On an existing feature branch, `gmr` reuses the branch and any commits ahead
  of the base branch. Uncommitted changes are committed there first.
- On the base branch, when there are no working-tree changes but the base
  branch has commits not yet pushed to `origin` (e.g. after `gmr -m` committed
  there), `gmr` moves those commits to a new branch instead of erroring, then
  opens the MR/PR and resets local base to `origin/<base>`. If there are also
  working-tree changes, the new commit and the previously unpushed ones are
  all included in the MR/PR, and the same base-branch reset happens after.
- `gmr -u` / `--update` downloads the latest stable GitHub release for the
  current OS/architecture, verifies its archive against `checksums.txt`, and
  replaces the running executable (following symlinks). It needs no repository,
  git, Go, hosting CLI, or AI key. Equal/newer versions are left unchanged;
  unversioned development builds are replaced. Use it alone, without `-c`,
  `-m`, `-s`, or `branch-name`. The installation directory must be writable.
  Windows keeps the previous executable as `.old` until the next update.
- `deploy` and `status` are reserved when they are the first argument.
- `gmr status` exits with status 1 when the newest run of any inspected ref has
  failed.
- `gmr -m` prints the generated message, then asks
  `Commit to '<current-branch>'? [Y/n/e(edit)] (n = print only):`. `y`/empty
  commits to the current branch (no new branch/push/MR/PR) and prints a
  "Next steps" block of push/MR-PR commands; `n` (default on non-TTY stdin,
  and any unrecognized answer) only prints; `e` opens `$EDITOR` first.
- `gmr -c` uses the same prompt (`Commit and push to '<current-branch>'?`)
  but pushes the current branch (`git push -u origin <branch>`) after the
  commit; it never creates a branch or MR/PR and needs no `gh`/`glab`. With a
  clean working tree it skips AI and pushes unpushed commits, or errors when
  the branch is up to date with `origin`. It warns on the base branch and
  rejects `-m`, `-s`, and `branch-name`.

## Build, test, and lint

Run the checks relevant to the change before reporting completion:

```bash
go build ./...
go test -race ./...
go vet ./...
gofmt -l .                 # must print nothing
golangci-lint run          # when golangci-lint is installed
```

CI uses Go 1.25, `go vet`, golangci-lint, race-enabled tests with coverage, and
a binary build smoke test.

## Runtime dependencies

- Go 1.25+ for building or installing from source.
- `git` for all repository operations.
- Authenticated `glab` for GitLab repositories or `gh` for GitHub repositories.
- At least one of `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, or `OPENAI_API_KEY`
  when AI output is required. Opening an MR/PR from an already committed feature
  branch and `gmr status` do not require an AI key; `gmr deploy` has a non-AI
  fallback.

## Environment variables

- Provider credentials: `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`.
- Model overrides: `GEMINI_MODEL` (default `gemini-flash-latest`),
  `ANTHROPIC_MODEL` (default `claude-sonnet-4-20250514`), and `OPENAI_MODEL`
  (default `gpt-4o-mini`).
- Endpoint overrides: `GEMINI_BASE_URL`, `ANTHROPIC_BASE_URL`, and
  `OPENAI_BASE_URL` (the latter supports OpenAI-compatible proxies such as
  LiteLLM).
- `GMR_PROVIDERS`: comma-separated fallback order; default
  `gemini,claude,openai` (`anthropic` is accepted as an alias for `claude`).
- `GMR_COMMIT_STYLE`: `human` (default) or `conventional`.
- `GMR_MAIN_BRANCH`: base branch override; otherwise detected from
  `origin/HEAD`, then local `main`/`master`.
- `GMR_MAX_DIFF`: maximum diff or log lines sent to AI; default `500`.
- `GMR_TAG_PREFIX`: prefix for the first generated release tag; default `v`.
- `EDITOR`: editor for the interactive edit choice; default `vim`.
- `NO_COLOR`: disables ANSI colors.

## Security and privacy

- Commit generation, including `gmr -m` and `gmr -c`, runs `git add -A` and sends the diff
  stat plus up to `GMR_MAX_DIFF` diff lines to the selected AI endpoint.
- Release generation may send commit subjects and bodies since the previous tag.
- `GMR_MAX_DIFF` is a line limit, not an opt-out switch; zero or invalid values
  restore the default. For sensitive changes, commit manually instead of using
  AI generation.
- Treat API keys as secrets and point `*_BASE_URL` variables only at trusted
  endpoints. Never log keys or include them in errors, test fixtures, or docs.

## Implementation and testing conventions

- Keep AI providers stateless and pass the complete prompt to `Generate`.
- AI provider tests use `httptest` and temporarily replace `ai.HTTPClient`,
  restoring it afterward.
- Route git operations through `git.Runner`; route CI CLI queries through
  `ci.Runner`. Use fakes in unit tests instead of invoking real repositories or
  remote services.
- Put pure helper tests beside their package in `*_test.go`. Extend command tests
  in `cmd/gmr/*_test.go` when parsing or orchestration behavior changes.
- Preserve the output contract: `ui.Log`, `ui.OK`, `ui.Warn`, and `ui.Errf` write
  to stderr; `gmr -m` writes only the commit message to stdout.
- Do not make network-backed end-to-end tests part of the default unit suite.

## Required accompanying changes

- Bump `internal/version/version.go` for every repository change: patch for
  fixes/docs/refactors, minor for features, major for breaking changes.
- Add a dated version section to `CHANGELOG.md`, using the applicable
  `Added`/`Changed`/`Fixed`/`Removed` headings.
- Update tests for changed behavior and `README.md` for user-visible flags,
  environment variables, installation steps, or workflows.
- Keep unrelated user changes intact and keep generated build artifacts out of
  commits.

## Releases

Pushing a `vX.Y.Z` tag triggers `.github/workflows/release.yml`. It tests the
tag, builds Linux, macOS, and Windows archives for amd64 and arm64, generates
`checksums.txt`, and creates the GitHub Release. Because the workflow owns GitHub
Release creation for this repository, use `gmr deploy --no-release` when cutting
a release here.

Before tagging, confirm that `internal/version/version.go` and `CHANGELOG.md`
match the tag and that the full build/test/lint checks pass.

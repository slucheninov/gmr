# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0] - 2026-03-29

### Added
- Initial gmr script: stages changes, generates AI commit message, creates branch, commits, opens GitLab MR
- Commit message generation via Gemini API (primary) and Claude API (fallback)
- Manual input fallback when both APIs are unavailable
- Interactive accept/reject/edit flow for generated commit messages
- Help (`-h`, `--help`) and version (`-v`, `--version`) options
- Install script with `~/.gmr/bin` directory and `/usr/local/bin` symlink
- Truncation detection for Gemini and Claude API responses
- Configurable via environment variables: `GMR_MAIN_BRANCH`, `GMR_GEMINI_MODEL`, `GMR_ANTHROPIC_MODEL`, `GMR_MAX_DIFF`

### Changed
- UI messages translated to English for log/error output
- Logging functions (`log`, `ok`, `warn`, `err`) redirect to stderr
- `--squash-before-merge` option added to `glab mr create`
- Max output tokens increased for both Gemini and Claude APIs

## [0.6.0] - 2026-04-25

### Changed
- **Full rewrite from Bash to Go.** `gmr` is now a single statically-linked Go binary built from `./cmd/gmr` instead of a Bash script. Functionality is preserved (Gemini → Claude → OpenAI → manual fallback, GitLab/GitHub auto-detection, MR/PR creation, auto-merge for GitHub, return-to-main with cleanup), but the implementation is split into testable packages: `internal/ai`, `internal/git`, `internal/platform`, `internal/commit`, `internal/ui`, `internal/version`.
- Distribution moved from raw script (`releases/latest/download/gmr` + `install.sh`) to per-OS/arch tarballs (`gmr-vX.Y.Z-{linux,darwin}-{amd64,arm64}.tar.gz`) with a single `checksums.txt`. `go install github.com/slucheninov/gmr/cmd/gmr@latest` is also supported.
- CI (`.github/workflows/ci.yml`) now runs `go vet`, `golangci-lint`, `go test -race -coverprofile`, and a `go build` smoke test on Go 1.25 (modeled on `tentens-tech/gomcrouter`). Release workflow now triggers on `v*` tags, runs tests, cross-compiles binaries via a build matrix, and uploads them to a GitHub Release with combined SHA-256 checksums.
- `--message` mode now writes the generated commit message to `stdout` (logs go to `stderr`), so `gmr -m` is pipe-friendly.

### Added
- Go test suite covering platform detection (`Detect`, `GitLabProjectPath`), commit-message helpers (`Title`, `Body`, `MRDescription`), main-branch resolution (`GMR_MAIN_BRANCH` → `origin/HEAD` → `main`/`master`), diff truncation (`LimitLines`), and AI providers via `httptest` (success, API-error payloads, truncation handling for Gemini `MAX_TOKENS`, Claude `max_tokens`, OpenAI `length`).
- `NO_COLOR` env var disables ANSI colors in log output.
- `Version` is now overridable at build time via `-ldflags "-X github.com/slucheninov/gmr/internal/version.Version=..."`, used by both CI and the release pipeline.

### Removed
- `gmr` Bash script, `install.sh`, `scripts/extract-release-notes.sh`, `tests/*.bats`, and the `bats`/`shellcheck` CI jobs — superseded by the Go implementation and Go-native tooling.
- `jq` and `curl` runtime dependencies (the Go binary speaks HTTP and JSON natively).

## [0.5.0] - 2026-04-24

### Added
- GitHub Actions workflow `.github/workflows/release.yml` that automatically creates a GitHub Release when `GMR_VERSION` in `gmr` is bumped on `master`/`main`: it tags `vX.Y.Z`, extracts release notes for that version from `CHANGELOG.md` (falling back to `[Unreleased]`), builds `gmr-X.Y.Z.tar.gz` / `gmr-X.Y.Z.zip` archives bundling `gmr`, `install.sh`, `README.md`, `LICENSE`, `CHANGELOG.md`, generates a `gmr-X.Y.Z.sha256` checksums file, and attaches `gmr`, `install.sh`, both archives and the checksums file as release assets. Also supports manual `workflow_dispatch`.
- Bumped `actions/checkout` from `@v4` to `@v5` in CI and release workflows (v5 ships with a native Node.js 24 runtime, addressing the GitHub Actions Node.js 20 deprecation: forced default on 2026-06-02, removal on 2026-09-16). Removed the now-unnecessary `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24` env override.
- `install.sh` now installs from GitHub Releases by default (`releases/latest/download/gmr`) with automatic fallback to raw branch download. New env vars: `GMR_INSTALL_FROM` (`release` (default) | `branch`) and `GMR_INSTALL_VERSION` (release tag, e.g. `v0.5.0`, default: `latest`).
- Test suite under `tests/` powered by [bats-core](https://github.com/bats-core/bats-core): covers `gmr` CLI flags (`-h`/`--help`/`-v`/`--version`, version constant consistency), `install.sh` flags (help, unknown-flag rejection), and `scripts/extract-release-notes.sh` (version section extraction, `[Unreleased]` fallback, default-message fallback, usage/error exit codes).
- `scripts/extract-release-notes.sh` extracted from the inline `awk` in `release.yml` so the same logic is shared between the workflow and tests.
- `.github/workflows/ci.yml` runs `bash -n`, `shellcheck` (blocking on `install.sh` and `scripts/*.sh`, informational on `gmr`) and bats on pushes to `master`/`main` and on pull requests.

### Fixed
- Gracefully handle GitHub repositories where auto-merge is disabled: `gh pr merge --auto --squash` failures (e.g. `Auto merge is not allowed for this repository`) no longer abort the script — a warning is shown and `gmr` continues to return to the main branch

### Changed
- GitLab fallback MR description no longer includes a `## Changes` section with staged diff stat; when the commit message has no body, `gmr` now generates only a short `## Summary` from the commit title
- AI-generated commit messages now use `type: description` without an optional scope like `feat(detection): ...`

### Fixed
- Installer now defaults to downloading from `master` (with fallback to `main`) and supports override via `GMR_INSTALL_BRANCH`, so it works correctly with repositories that keep `master` as default while remaining compatible with `main`
- Main branch detection now auto-resolves from `origin/HEAD` when `GMR_MAIN_BRANCH` is not set, so repositories with `main` no longer fail with `Current branch is 'main', not 'master'`
- GitLab MR creation no longer passes `--fill` together with explicit `--title`/`--description`, avoiding the `glab` error `Usage of --title and --description overrides --fill`
- GitLab MR creation now always passes a non-empty description to `glab mr create`: when the commit message has no body, `gmr` auto-generates an MR description from the commit title and staged diff stat, avoiding the interactive description prompt
- `gmr` no longer exits right after printing `Platform` and `Branch` when collecting the staged diff: the diff preview no longer uses `head` inside a `pipefail` pipeline, which previously caused a silent exit with `SIGPIPE`
- GitLab MR creation now passes repository, source branch, target branch, and title explicitly to `glab mr create`, avoiding 404 errors when `glab` mis-detects MR parameters in non-interactive mode
- `gmr` now checks `glab`/`gh` API authentication before creating MR/PR and shows a clear login hint instead of failing later with a confusing 404 on private repositories

### Changed
- GitLab repository reference for `glab mr create` now uses the canonical `group/project` path parsed from `origin`, instead of the SSH remote URL

## [0.4.1] - 2026-04-13

### Fixed
- GitLab MR creation failed with 404 because branch was not pushed to remote before calling `glab mr create`; now pushes branch first

### Added
- GitHub support: auto-detects platform (GitLab/GitHub) from `origin` remote URL
- Uses `gh pr create` for GitHub repos, `glab mr create` for GitLab repos
- OpenAI (ChatGPT) support as third fallback: Gemini → Claude → OpenAI → manual
- `OPENAI_API_KEY` env var and `GMR_OPENAI_MODEL` config (default: `gpt-4o-mini`)
- `-m` / `--message` flag: generate commit message only, without creating branch, commit, or MR/PR

### Changed
- Gemini API key moved from URL query parameter to `x-goog-api-key` header (recommended approach)
- Default Gemini model changed to `gemini-flash-latest`

### Fixed
- Script now returns to main branch even if MR/PR creation fails (via `trap EXIT`)
- Gemini truncation no longer causes full failure — first line of response is used as commit message
- Claude truncation no longer causes full failure — same fix applied
- Truncation warning now includes diff size for diagnostics

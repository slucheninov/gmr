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

## [Unreleased]

### Changed
- GitLab fallback MR description no longer includes a `## Changes` section with staged diff stat; when the commit message has no body, `gmr` now generates only a short `## Summary` from the commit title
- AI-generated commit messages now use `type: description` without an optional scope like `feat(detection): ...`

### Fixed
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

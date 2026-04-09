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

### Added
- GitHub support: auto-detects platform (GitLab/GitHub) from `origin` remote URL
- Uses `gh pr create` for GitHub repos, `glab mr create` for GitLab repos

### Changed
- Gemini API key moved from URL query parameter to `x-goog-api-key` header (recommended approach)
- Default Gemini model changed to `gemini-flash-latest`

### Fixed
- Gemini truncation no longer causes full failure — first line of response is used as commit message
- Claude truncation no longer causes full failure — same fix applied
- Truncation warning now includes diff size for diagnostics

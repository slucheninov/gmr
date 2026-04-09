# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`gmr` (Git Merge Request) is a single-file Bash script that automates the GitLab merge request workflow. It stages changes, generates a commit message via the Claude API, creates a branch, commits, and opens a GitLab MR — all in one command.

## Usage

```bash
./gmr [branch-name]    # branch-name defaults to auto/YYYYMMDD-HHMMSS
```

## Dependencies

- `glab` (GitLab CLI), `jq`, `curl`
- `GEMINI_API_KEY` and/or `ANTHROPIC_API_KEY` (at least one required)

## Configuration (env vars)

- `GMR_MAIN_BRANCH` — base branch (default: `master`)
- `GMR_GEMINI_MODEL` — Gemini model (default: `gemini-flash-latest`)
- `GMR_ANTHROPIC_MODEL` — Anthropic model (default: `claude-sonnet-4-20250514`)
- `GMR_MAX_DIFF` — max diff lines sent to API (default: `500`)

## Architecture

Single script (`gmr`), sequential flow:
1. Pre-checks (tools installed, API key set, on main branch, changes exist)
2. `git add -A` + generate commit message: Gemini (default) → Claude (fallback) → manual input
3. Create branch, commit, `glab mr create --fill`, return to main branch

## Rules for changes

- **Version**: always bump `GMR_VERSION` in `gmr` (semver: patch for fixes, minor for features, major for breaking changes)
- **Changelog**: always update `CHANGELOG.md` — add entry under `[Unreleased]` section (Added/Changed/Fixed/Removed)

## Notes

- UI messages are in Ukrainian
- Fallback to manual input if API call fails or response can't be parsed

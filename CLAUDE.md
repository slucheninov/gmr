# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`gmr` (Git Merge Request) is a single-file Bash script that automates the merge request / pull request workflow. It stages changes, generates a commit message via AI (Gemini / Claude), creates a branch, commits, and opens a GitLab MR or GitHub PR — all in one command. Platform is auto-detected from the `origin` remote URL.

## Usage

```bash
./gmr [branch-name]    # branch-name defaults to auto/YYYYMMDD-HHMMSS
```

## Dependencies

- `glab` (GitLab CLI) or `gh` (GitHub CLI), `jq`, `curl`
- `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, and/or `OPENAI_API_KEY` (at least one required)

## Configuration (env vars)

- `GMR_MAIN_BRANCH` — base branch (default: `master`)
- `GMR_GEMINI_MODEL` — Gemini model (default: `gemini-flash-latest`)
- `GMR_ANTHROPIC_MODEL` — Anthropic model (default: `claude-sonnet-4-20250514`)
- `GMR_OPENAI_MODEL` — OpenAI model (default: `gpt-4o-mini`)
- `GMR_MAX_DIFF` — max diff lines sent to API (default: `500`)

## Architecture

Single script (`gmr`), sequential flow:
1. Pre-checks (tools installed, API key set, on main branch, changes exist)
2. `git add -A` + generate commit message: Gemini → Claude → OpenAI → manual input
3. Create branch, commit, open MR/PR (`glab mr create` or `gh pr create`), return to main branch

## Rules for changes

- **Version**: always bump `GMR_VERSION` in `gmr` (semver: patch for fixes, minor for features, major for breaking changes)
- **Changelog**: always update `CHANGELOG.md` — add entry under `[Unreleased]` section (Added/Changed/Fixed/Removed)
- **README**: update `README.md` if changes affect user-facing info (new features, changed defaults, new env vars, new dependencies, changed workflow)

## Notes

- UI messages are in Ukrainian
- Fallback to manual input if API call fails or response can't be parsed

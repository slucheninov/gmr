#!/usr/bin/env bash
# Extract release notes for a given version from a Keep-a-Changelog-style file.
# Usage: extract-release-notes.sh <version> [changelog-path]
# Behavior:
#   - Prints lines between `## [<version>]` and the next `## [` heading.
#   - If no section for <version>, falls back to `## [Unreleased]`.
#   - If still empty, prints `Release v<version>`.
# Exit codes: 0 on success, 2 on bad usage, 3 if changelog file missing.
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "Usage: $0 <version> [changelog-path]" >&2
  exit 2
fi

VERSION="$1"
CHANGELOG="${2:-CHANGELOG.md}"

if [[ ! -f "$CHANGELOG" ]]; then
  echo "Changelog file not found: $CHANGELOG" >&2
  exit 3
fi

notes="$(awk -v ver="$VERSION" '
  $0 ~ "^## \\[" ver "\\]" { capture=1; next }
  capture && /^## \[/ { exit }
  capture { print }
' "$CHANGELOG")"

if [[ -z "${notes//[[:space:]]/}" ]]; then
  notes="$(awk '
    /^## \[Unreleased\]/ { capture=1; next }
    capture && /^## \[/ { exit }
    capture { print }
  ' "$CHANGELOG")"
fi

if [[ -z "${notes//[[:space:]]/}" ]]; then
  notes="Release v${VERSION}"
fi

printf '%s\n' "$notes"

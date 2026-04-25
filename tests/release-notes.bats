#!/usr/bin/env bats

# Tests for scripts/extract-release-notes.sh.

setup() {
  REPO_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  SCRIPT="$REPO_ROOT/scripts/extract-release-notes.sh"
  TMPDIR_TEST="$(mktemp -d)"
}

teardown() {
  rm -rf "$TMPDIR_TEST"
}

write_changelog() {
  cat > "$TMPDIR_TEST/CHANGELOG.md" <<'EOF'
# Changelog

## [Unreleased]

### Added
- Pending feature A

## [1.2.0] - 2026-04-01

### Added
- Feature X
- Feature Y

### Fixed
- Bug Z

## [1.1.0] - 2026-03-01

### Added
- Old feature

EOF
}

@test "extracts notes for an existing version" {
  write_changelog
  run "$SCRIPT" 1.2.0 "$TMPDIR_TEST/CHANGELOG.md"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Feature X"* ]]
  [[ "$output" == *"Feature Y"* ]]
  [[ "$output" == *"Bug Z"* ]]
  [[ "$output" != *"Old feature"* ]]
  [[ "$output" != *"Pending feature A"* ]]
}

@test "stops at next version heading (no bleed-through)" {
  write_changelog
  run "$SCRIPT" 1.1.0 "$TMPDIR_TEST/CHANGELOG.md"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Old feature"* ]]
  [[ "$output" != *"Feature X"* ]]
}

@test "falls back to [Unreleased] when version section missing" {
  write_changelog
  run "$SCRIPT" 9.9.9 "$TMPDIR_TEST/CHANGELOG.md"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Pending feature A"* ]]
  [[ "$output" != *"Feature X"* ]]
}

@test "falls back to default message when no [Unreleased] either" {
  cat > "$TMPDIR_TEST/CHANGELOG.md" <<'EOF'
# Changelog

## [1.0.0] - 2026-01-01
- something
EOF
  run "$SCRIPT" 9.9.9 "$TMPDIR_TEST/CHANGELOG.md"
  [ "$status" -eq 0 ]
  [ "$output" = "Release v9.9.9" ]
}

@test "fails with usage error when no args given" {
  run "$SCRIPT"
  [ "$status" -eq 2 ]
  [[ "$output" == *"Usage:"* ]]
}

@test "fails when changelog file does not exist" {
  run "$SCRIPT" 1.0.0 "$TMPDIR_TEST/nope.md"
  [ "$status" -eq 3 ]
  [[ "$output" == *"not found"* ]]
}

@test "works against the real repo CHANGELOG.md (smoke)" {
  run "$SCRIPT" 0.4.1 "$REPO_ROOT/CHANGELOG.md"
  [ "$status" -eq 0 ]
  [[ "$output" == *"GitHub support"* ]]
}

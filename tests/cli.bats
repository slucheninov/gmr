#!/usr/bin/env bats

# Tests for gmr and install.sh CLI surface (flags only — no git/network calls).

setup() {
  REPO_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  GMR="$REPO_ROOT/gmr"
  INSTALL="$REPO_ROOT/install.sh"
}

# ── gmr ─────────────────────────────────────────────────────────────────

@test "gmr -h prints usage and exits 0" {
  run "$GMR" -h
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: gmr"* ]]
  [[ "$output" == *"--message"* ]]
}

@test "gmr --help prints usage and exits 0" {
  run "$GMR" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: gmr"* ]]
}

@test "gmr -v prints semver version" {
  run "$GMR" -v
  [ "$status" -eq 0 ]
  [[ "$output" =~ ^gmr\ [0-9]+\.[0-9]+\.[0-9]+$ ]]
}

@test "gmr --version prints semver version" {
  run "$GMR" --version
  [ "$status" -eq 0 ]
  [[ "$output" =~ ^gmr\ [0-9]+\.[0-9]+\.[0-9]+$ ]]
}

@test "gmr version matches GMR_VERSION constant in script" {
  expected="$(grep -E '^GMR_VERSION=' "$GMR" | head -n1 | sed -E 's/^GMR_VERSION="?([^"]+)"?/\1/')"
  run "$GMR" --version
  [ "$status" -eq 0 ]
  [ "$output" = "gmr $expected" ]
}

# ── install.sh ──────────────────────────────────────────────────────────

@test "install.sh -h prints usage and exits 0" {
  run "$INSTALL" -h
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: install.sh"* ]]
  [[ "$output" == *"GMR_INSTALL_FROM"* ]]
  [[ "$output" == *"GMR_INSTALL_VERSION"* ]]
}

@test "install.sh --help prints usage and exits 0" {
  run "$INSTALL" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: install.sh"* ]]
}

@test "install.sh rejects unknown flags" {
  run "$INSTALL" --bogus
  [ "$status" -ne 0 ]
  [[ "$output" == *"Unknown option"* ]]
}

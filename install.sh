#!/usr/bin/env bash
set -euo pipefail

REPO="slucheninov/gmr"
DEFAULT_BRANCH="${GMR_INSTALL_BRANCH:-master}"
FALLBACK_BRANCH="main"
RAW_BASE_URL="https://raw.githubusercontent.com/${REPO}"
RELEASE_BASE_URL="https://github.com/${REPO}/releases"
INSTALL_FROM="${GMR_INSTALL_FROM:-release}"   # release | branch
INSTALL_VERSION="${GMR_INSTALL_VERSION:-latest}"
GMR_HOME="$HOME/.gmr"
GMR_BIN="$GMR_HOME/bin/gmr"
LINK_DIR="${GMR_INSTALL_DIR:-/usr/local/bin}"
FORCE=false

# ── Colors ─────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${CYAN}▸${NC} $1"; }
ok()   { echo -e "${GREEN}✔${NC} $1"; }
warn() { echo -e "${YELLOW}⚠${NC} $1"; }
err()  { echo -e "${RED}✖${NC} $1" >&2; exit 1; }

# ── Parse args ─────────────────────────────────────────────────────────
for arg in "$@"; do
  case "$arg" in
    -f|--force) FORCE=true ;;
    -h|--help)
      cat <<EOF
Usage: install.sh [-f|--force]
  -f, --force   Force reinstall even if already installed

Environment variables:
  GMR_INSTALL_FROM     Source: 'release' (default) or 'branch'
  GMR_INSTALL_VERSION  Release tag to install, e.g. v0.5.0 (default: latest)
  GMR_INSTALL_BRANCH   Preferred branch for branch-mode / release fallback (default: master)
  GMR_INSTALL_DIR      Symlink directory (default: /usr/local/bin)
EOF
      exit 0
      ;;
    *) err "Unknown option: $arg" ;;
  esac
done

# ── Check if already installed ─────────────────────────────────────────
if [[ -f "$GMR_BIN" ]] && [[ "$FORCE" == false ]]; then
  warn "gmr is already installed at $GMR_BIN"
  warn "Use -f or --force to reinstall"
  exit 0
fi

# ── Detect download tool ──────────────────────────────────────────────
if command -v curl >/dev/null 2>&1; then
  download() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  download() { wget -qO- "$1"; }
else
  err "curl or wget not found"
fi

# ── Download ───────────────────────────────────────────────────────────
tmpfile=$(mktemp)
source_label=""

download_from_release() {
  local version_path
  if [[ "$INSTALL_VERSION" == "latest" ]]; then
    version_path="latest/download"
  else
    version_path="download/${INSTALL_VERSION}"
  fi
  local url="${RELEASE_BASE_URL}/${version_path}/gmr"
  if download "$url" > "$tmpfile" 2>/dev/null && [[ -s "$tmpfile" ]]; then
    source_label="release ${INSTALL_VERSION}"
    return 0
  fi
  return 1
}

download_from_branch() {
  local branch tried_branch="" url
  local candidates=("$DEFAULT_BRANCH" "$FALLBACK_BRANCH")
  for branch in "${candidates[@]}"; do
    [[ -n "$branch" ]] || continue
    [[ "$branch" == "$tried_branch" ]] && continue
    tried_branch="$branch"
    url="${RAW_BASE_URL}/${branch}/gmr"
    if download "$url" > "$tmpfile" 2>/dev/null && [[ -s "$tmpfile" ]]; then
      source_label="branch ${branch}"
      return 0
    fi
  done
  return 1
}

log "Downloading gmr (source: ${INSTALL_FROM}, version: ${INSTALL_VERSION})..."
if [[ "$INSTALL_FROM" == "release" ]]; then
  download_from_release || {
    warn "Release download failed, falling back to branch ${DEFAULT_BRANCH}"
    download_from_branch || err "Failed to download gmr from release '${INSTALL_VERSION}' or branches '${DEFAULT_BRANCH}'/'${FALLBACK_BRANCH}'"
  }
else
  download_from_branch || err "Failed to download gmr from '${DEFAULT_BRANCH}' or '${FALLBACK_BRANCH}'"
fi
ok "Downloaded from ${source_label}"

# ── Install to ~/.gmr/bin ─────────────────────────────────────────────
mkdir -p "$GMR_HOME/bin"
mv "$tmpfile" "$GMR_BIN"
chmod +x "$GMR_BIN"
ok "Installed to $GMR_BIN"

# ── Symlink to /usr/local/bin (or custom dir) ─────────────────────────
log "Creating symlink in $LINK_DIR..."
if [[ -w "$LINK_DIR" ]]; then
  ln -sf "$GMR_BIN" "$LINK_DIR/gmr"
else
  sudo ln -sf "$GMR_BIN" "$LINK_DIR/gmr"
fi

ok "gmr installed! Verify: gmr --help"

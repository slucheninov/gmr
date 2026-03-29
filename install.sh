#!/usr/bin/env bash
set -euo pipefail

REPO="slucheninov/gmr"
BRANCH="master"
RAW_URL="https://raw.githubusercontent.com/${REPO}/${BRANCH}/gmr"
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
      echo "Usage: install.sh [-f|--force]"
      echo "  -f, --force   Force reinstall even if already installed"
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
log "Downloading gmr..."
tmpfile=$(mktemp)
download "$RAW_URL" > "$tmpfile" || err "Failed to download gmr"

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

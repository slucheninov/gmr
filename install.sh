#!/usr/bin/env bash
set -euo pipefail

REPO="slucheninov/gmr"
BRANCH="master"
RAW_URL="https://raw.githubusercontent.com/${REPO}/${BRANCH}/gmr"
INSTALL_DIR="${GMR_INSTALL_DIR:-/usr/local/bin}"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

log() { echo -e "${CYAN}▸${NC} $1"; }
ok()  { echo -e "${GREEN}✔${NC} $1"; }
err() { echo -e "${RED}✖${NC} $1" >&2; exit 1; }

# Detect download tool
if command -v curl >/dev/null 2>&1; then
  download() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  download() { wget -qO- "$1"; }
else
  err "curl або wget не знайдено"
fi

log "Завантажую gmr..."
tmpfile=$(mktemp)
download "$RAW_URL" > "$tmpfile" || err "Не вдалось завантажити gmr"

log "Встановлюю в ${INSTALL_DIR}/gmr..."
if [[ -w "$INSTALL_DIR" ]]; then
  mv "$tmpfile" "${INSTALL_DIR}/gmr"
  chmod +x "${INSTALL_DIR}/gmr"
else
  sudo mv "$tmpfile" "${INSTALL_DIR}/gmr"
  sudo chmod +x "${INSTALL_DIR}/gmr"
fi

ok "gmr встановлено! Перевір: gmr --help"

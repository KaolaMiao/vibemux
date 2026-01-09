#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-UgOrange/vibemux}"
RELEASE_TAG="${RELEASE_TAG:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BIN_NAME="vibemux"

uname_s="$(uname -s)"
case "$uname_s" in
  Darwin) OS="darwin" ;;
  Linux) OS="linux" ;;
  *)
    echo "Unsupported OS: $uname_s. Use WSL on Windows." >&2
    exit 1
    ;;
esac

uname_m="$(uname -m)"
case "$uname_m" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported arch: $uname_m" >&2
    exit 1
    ;;
esac

ASSET="${BIN_NAME}-${OS}-${ARCH}"
if [ "$RELEASE_TAG" = "latest" ]; then
  URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"
else
  URL="https://github.com/${REPO}/releases/download/${RELEASE_TAG}/${ASSET}"
fi

tmp="$(mktemp)"
cleanup() { rm -f "$tmp"; }
trap cleanup EXIT

echo "Downloading ${URL}"
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "$tmp"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tmp" "$URL"
else
  echo "curl or wget is required." >&2
  exit 1
fi

chmod +x "$tmp"
mkdir -p "$INSTALL_DIR"
mv "$tmp" "$INSTALL_DIR/$BIN_NAME"

echo "Installed ${BIN_NAME} to ${INSTALL_DIR}/${BIN_NAME}"
if ! command -v "$BIN_NAME" >/dev/null 2>&1; then
  echo "Add ${INSTALL_DIR} to your PATH to run '${BIN_NAME}'."
fi

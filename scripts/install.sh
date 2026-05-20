#!/bin/sh
# One CLI installer — curl one-liner:
#   curl -fsSL https://raw.githubusercontent.com/elydelva/one/main/scripts/install.sh | sh

set -e

REPO="elydelva/one"
INSTALL_DIR="${ONE_INSTALL_DIR:-/usr/local/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version." >&2
  exit 1
fi

ARCHIVE="one_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$ARCHIVE"

echo "Installing one $VERSION for $OS/$ARCH..."
TMPDIR=$(mktemp -d)
curl -fsSL "$URL" | tar -xz -C "$TMPDIR"
install -m 755 "$TMPDIR/one" "$INSTALL_DIR/one"
rm -rf "$TMPDIR"

echo "Installed: $(command -v one)"
one --version

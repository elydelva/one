#!/bin/sh
# One CLI installer — curl one-liner:
#   curl -fsSL https://raw.githubusercontent.com/elydelva/one/main/scripts/install.sh | sh
#
# Verifies the SHA256 of the downloaded archive against the release
# checksums.txt before installing.

set -eu

REPO="elydelva/one"
INSTALL_DIR="${ONE_INSTALL_DIR:-/usr/local/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

VERSION="${ONE_VERSION:-}"
if [ -z "$VERSION" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
fi

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version." >&2
  exit 1
fi

VERSION_NUM="${VERSION#v}"
ARCHIVE="one_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/$REPO/releases/download/$VERSION"
URL="$BASE_URL/$ARCHIVE"
SUMS_URL="$BASE_URL/checksums.txt"

echo "Installing one $VERSION for $OS/$ARCH..."
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL -o "$TMPDIR/$ARCHIVE" "$URL"
curl -fsSL -o "$TMPDIR/checksums.txt" "$SUMS_URL"

# Verify SHA256. Use shasum (macOS) or sha256sum (linux).
EXPECTED=$(grep " $ARCHIVE\$" "$TMPDIR/checksums.txt" | awk '{print $1}')
if [ -z "$EXPECTED" ]; then
  echo "Checksum for $ARCHIVE missing from checksums.txt" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL=$(sha256sum "$TMPDIR/$ARCHIVE" | awk '{print $1}')
else
  ACTUAL=$(shasum -a 256 "$TMPDIR/$ARCHIVE" | awk '{print $1}')
fi
if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "Checksum mismatch:" >&2
  echo "  expected: $EXPECTED" >&2
  echo "  actual:   $ACTUAL" >&2
  exit 1
fi
echo "Checksum OK ($ACTUAL)"

tar -xz -C "$TMPDIR" -f "$TMPDIR/$ARCHIVE"
install -m 755 "$TMPDIR/one" "$INSTALL_DIR/one"

echo "Installed: $(command -v one)"
one --version

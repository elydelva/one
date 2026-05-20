#!/usr/bin/env bash
# Build a One CLI handler from a bundled JS entrypoint using Javy.
# Usage: ./scripts/build-wasm.sh dist/handler.js dist/handler.wasm
set -euo pipefail

src="${1:-dist/handler.js}"
out="${2:-dist/handler.wasm}"

if ! command -v javy >/dev/null 2>&1; then
  echo "javy not found in PATH — install from https://github.com/bytecodealliance/javy" >&2
  exit 1
fi

javy compile "$src" -o "$out"
echo "built $out ($(stat -f%z "$out" 2>/dev/null || stat -c%s "$out") bytes)"

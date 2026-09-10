#!/usr/bin/env bash
# Role: Fetch JetBrains Mono Regular (400) woff2 for marketing monospace surfaces.
# Env: JETBRAINS_MONO_WOFF2_URL (optional override)
# Verify:
#   bash scripts/ops/bundle_marketing_jetbrains_mono.sh
#   test -f deploy/marketing/fonts/jetbrains-mono/JetBrainsMono-Regular.woff2
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

OUT_DIR="$ROOT/deploy/marketing/fonts/jetbrains-mono"
OUT_FILE="$OUT_DIR/JetBrainsMono-Regular.woff2"
SRC_URL="${JETBRAINS_MONO_WOFF2_URL:-https://github.com/JetBrains/JetBrainsMono/raw/v2.304/fonts/webfonts/JetBrainsMono-Regular.woff2}"

mkdir -p "$OUT_DIR"

if [[ -f "$OUT_FILE" ]]; then
  printf 'bundle-marketing-jetbrains-mono: up to date %s\n' "$OUT_FILE"
  exit 0
fi

tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT

if ! curl -fsSL "$SRC_URL" -o "$tmp_file"; then
  printf 'bundle-marketing-jetbrains-mono: ERROR: failed to download %s\n' "$SRC_URL" >&2
  exit 1
fi

if [[ ! -s "$tmp_file" ]]; then
  printf 'bundle-marketing-jetbrains-mono: ERROR: empty download from %s\n' "$SRC_URL" >&2
  exit 1
fi

mv "$tmp_file" "$OUT_FILE"
trap - EXIT

printf 'bundle-marketing-jetbrains-mono: wrote %s\n' "$OUT_FILE"

#!/usr/bin/env bash
# Role: Optional content-diff drill for vision-agent residual risk (T28).
# Env: CONTENT_DIFF_DRILL_URL, CONTENT_DIFF_PHASH=1 enables sha256 body hash compare (not pixel pHash).
# Verify: bash scripts/test/edge/content_diff_drill.sh --holdout
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

URL="${CONTENT_DIFF_DRILL_URL:-}"
OUT_DIR="${CONTENT_DIFF_DRILL_ARTIFACT_DIR:-$ROOT/var/ci/content_diff_drill}"
PHASH_MODE="${CONTENT_DIFF_PHASH:-0}"

log() { printf 'content_diff_drill: %s\n' "$*" >&2; }
die() {
  printf 'content_diff_drill: ERROR: %s\n' "$*" >&2
  exit 1
}

holdout_selftest() {
  mkdir -p "$OUT_DIR"
  printf '<html><body>fixture-a</body></html>' > "$OUT_DIR/fixture_a.html"
  printf '<html><body>fixture-b</body></html>' > "$OUT_DIR/fixture_b.html"
  hash_a="$(sha256sum "$OUT_DIR/fixture_a.html" | awk '{print $1}')"
  hash_b="$(sha256sum "$OUT_DIR/fixture_b.html" | awk '{print $1}')"
  if [[ "$hash_a" == "$hash_b" ]]; then
    die "holdout: distinct fixtures must differ"
  fi
  log "holdout OK hash_a=${hash_a:0:12} hash_b=${hash_b:0:12}"
}

if [[ "${1:-}" == "--holdout" ]]; then
  holdout_selftest
  exit 0
fi

if [[ -z "$URL" ]]; then
  log "skip live drill: set CONTENT_DIFF_DRILL_URL"
  exit 0
fi

mkdir -p "$OUT_DIR"
body="$OUT_DIR/body.html"
curl -sS -L --max-time 30 -o "$body" "$URL"
hash="$(sha256sum "$body" | awk '{print $1}')"
printf '%s\n' "$hash" > "$OUT_DIR/body.sha256"
log "fetched url=$URL hash=${hash:0:16} phash_mode=$PHASH_MODE"

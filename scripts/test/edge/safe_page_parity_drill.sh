#!/usr/bin/env bash
# Role: Dual-egress safe-page parity drill (datacenter vs residential proxy).
# Execution context: Operator or CI with live tracker URL and optional residential proxy.
# Env knobs: SAFE_PAGE_PARITY_URL, SAFE_PAGE_PARITY_DC_PROXY, SAFE_PAGE_PARITY_RES_PROXY,
#   SAFE_PAGE_PARITY_MAX_DIFF (default 0), SAFE_PAGE_PARITY_ARTIFACT_DIR.
# Verify: bash scripts/test/edge/safe_page_parity_drill.sh --holdout
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

OUT_DIR="${SAFE_PAGE_PARITY_ARTIFACT_DIR:-$ROOT/var/ci/safe_page_parity}"
MAX_DIFF="${SAFE_PAGE_PARITY_MAX_DIFF:-0}"
CLICK_URL="${SAFE_PAGE_PARITY_URL:-}"
DC_PROXY="${SAFE_PAGE_PARITY_DC_PROXY:-}"
RES_PROXY="${SAFE_PAGE_PARITY_RES_PROXY:-}"
UA="${SAFE_PAGE_PARITY_UA:-Mozilla/5.0 (compatible; safe-page-parity-drill/1.0)}"

log() { printf 'safe_page_parity_drill: %s\n' "$*" >&2; }
die() {
  printf 'safe_page_parity_drill: ERROR: %s\n' "$*" >&2
  exit 1
}

mkdir -p "$OUT_DIR"

sha256_file() {
  sha256sum "$1" | awk '{print $1}'
}

count_scripts() {
  grep -oi '<script' "$1" 2> /dev/null | wc -l | tr -d ' '
}

fetch_leg() {
  local leg="$1"
  local proxy="${2:-}"
  local body_file="$OUT_DIR/body_${leg}.html"
  local meta_file="$OUT_DIR/meta_${leg}.txt"
  local curl_args=(
    -sS
    -L
    --max-redirs 10
    --max-time 30
    -A "$UA"
    -o "$body_file"
    -w '%{http_code}\n%{url_effective}\n%{num_redirects}\n'
  )
  if [[ -n "$proxy" ]]; then
    curl_args+=(--proxy "$proxy")
  fi
  curl_args+=("$CLICK_URL")
  local meta
  meta="$(curl "${curl_args[@]}")"
  local status effective_url redirects
  status="$(printf '%s' "$meta" | sed -n '1p')"
  effective_url="$(printf '%s' "$meta" | sed -n '2p')"
  redirects="$(printf '%s' "$meta" | sed -n '3p')"
  local body_hash script_count
  body_hash="$(sha256_file "$body_file")"
  script_count="$(count_scripts "$body_file")"
  printf '%s\n' "$status" "$effective_url" "$body_hash" "$script_count" "$redirects" > "$meta_file"
  printf '%s,%s,%s,%s,%s,%s\n' "$leg" "$status" "$effective_url" "$body_hash" "$script_count" "$redirects"
}

compare_legs() {
  local dc_meta="$OUT_DIR/meta_dc.txt"
  local res_meta="$OUT_DIR/meta_res.txt"
  local diff=0
  local fields=(status effective_url body_hash script_count redirects)
  local i
  for i in "${!fields[@]}"; do
    local dc_val res_val
    dc_val="$(sed -n "$((i + 1))p" "$dc_meta")"
    res_val="$(sed -n "$((i + 1))p" "$res_meta")"
    if [[ "$dc_val" != "$res_val" ]]; then
      diff=$((diff + 1))
      log "diff field=${fields[$i]} dc=${dc_val} res=${res_val}"
    fi
  done
  printf '%s' "$diff"
}

emit_fault_proof() {
  local status="$1"
  local diff="$2"
  local line
  line="fault_proof fault=safe_zone_parity status=${status} diff=${diff} max_diff=${MAX_DIFF} artifact_dir=${OUT_DIR} baseline_ok=true"
  printf '%s\n' "$line" | tee "$OUT_DIR/fault_proof.txt"
}

holdout_self_test() {
  log "holdout: identical bodies must pass"
  mkdir -p "$OUT_DIR/holdout"
  local hold="$OUT_DIR/holdout"
  printf '%s\n' '<html><head></head><body><script src="/a.js"></script></body></html>' > "$hold/a.html"
  cp "$hold/a.html" "$hold/b.html"
  printf '%s\n' '200' 'https://example.test/click' "$(sha256_file "$hold/a.html")" '1' '0' > "$hold/meta_dc.txt"
  printf '%s\n' '200' 'https://example.test/click' "$(sha256_file "$hold/b.html")" '1' '0' > "$hold/meta_res.txt"
  OUT_DIR="$hold" MAX_DIFF=0
  local d
  d="$(compare_legs)"
  [[ "$d" -eq 0 ]] || die "holdout identical bodies failed diff=${d}"

  log "holdout: body hash mismatch must fail policy"
  printf '%s\n' '<html><body>decoy</body></html>' > "$hold/b.html"
  sed -i '3s/.*/'"$(sha256_file "$hold/b.html")"'/' "$hold/meta_res.txt"
  d="$(compare_legs)"
  [[ "$d" -gt 0 ]] || die "holdout expected diff on hash mismatch"

  emit_fault_proof passed 0
  log "holdout: OK"
}

run_live_drill() {
  command -v curl > /dev/null 2>&1 || die "curl required"
  [[ -n "$CLICK_URL" ]] || die "SAFE_PAGE_PARITY_URL is required for live drill"

  log "fetch leg=dc proxy=${DC_PROXY:-direct}"
  fetch_leg dc "$DC_PROXY" | tee "$OUT_DIR/summary_dc.csv"
  if [[ -z "$RES_PROXY" ]]; then
    log "skip residential leg (SAFE_PAGE_PARITY_RES_PROXY unset)"
    emit_fault_proof skipped_partial 0
    log "partial drill complete (dc only)"
    return 0
  fi

  log "fetch leg=res proxy=${RES_PROXY}"
  fetch_leg res "$RES_PROXY" | tee "$OUT_DIR/summary_res.csv"
  {
    echo 'leg,status,final_url,body_sha256,script_count,redirect_depth'
    cat "$OUT_DIR/summary_dc.csv" "$OUT_DIR/summary_res.csv"
  } > "$OUT_DIR/summary.csv"

  local diff
  diff="$(compare_legs)"
  log "diff_count=${diff} max_diff=${MAX_DIFF}"
  if ((diff > MAX_DIFF)); then
    emit_fault_proof failed "$diff"
    die "policy breach: diff=${diff} > max_diff=${MAX_DIFF}"
  fi
  emit_fault_proof passed "$diff"
  log "ok"
}

case "${1:-}" in
  --holdout)
    holdout_self_test
    ;;
  "")
    if [[ "${SAFE_PAGE_PARITY_SKIP:-0}" = "1" ]]; then
      log "skip (SAFE_PAGE_PARITY_SKIP=1)"
      exit 0
    fi
    if [[ -z "$CLICK_URL" ]]; then
      log "skip live drill (SAFE_PAGE_PARITY_URL unset); running holdout"
      holdout_self_test
      exit 0
    fi
    run_live_drill
    ;;
  *)
    die "unknown arg: $1 (use --holdout)"
    ;;
esac

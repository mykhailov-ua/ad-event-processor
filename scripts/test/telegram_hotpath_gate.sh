#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

PROM_URL="${PROMETHEUS_URL:-http://127.0.0.1:9190}"
SESSION_DIR="${1:-}"

log() { printf 'telegram-hotpath-gate: %s\n' "$*"; }

log "fuzz smoke"
bash "$SCRIPTS/test/telegram_fuzz_smoke.sh"

log "alloc gate (telegram paths)"
go test -short -count=1 -run 'TgClick|ParseTgClick|DecodeTgBid|Check_zeroAlloc_localQuantaFullSkip' ./internal/ingestion/...

if [[ -n "$SESSION_DIR" && -d "$SESSION_DIR" ]]; then
  log "telegram gate report - $SESSION_DIR"
  go run ./cmd/load-report telegram "$SESSION_DIR" --prom "$PROM_URL"
else
  log "soak (optional stack must be up; set PREPARE=1 to bootstrap)"
  PREPARE="${PREPARE:-0}" bash "$SCRIPTS/test/telegram_hotpath_soak.sh"
  LATEST="$(ls -td "$ROOT"/var/load-test/tg-* 2> /dev/null | head -1 || true)"
  if [[ -n "$LATEST" ]]; then
    go run ./cmd/load-report telegram "$LATEST" --prom "$PROM_URL" || log "WARN: telegram gate skipped (no BPF/Prom)"
  fi
fi

log "PASS"

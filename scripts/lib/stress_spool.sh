#!/usr/bin/env bash

set -euo pipefail

# Role: Library: Stress CH spool for fault drills.
# Execution context: Sourced by CI, fault, and dev scripts; not a standalone gate.
# Invariants/contracts enforced: Helpers must not exit 0 on error paths when used as gate prerequisites.
# Verify: source scripts/lib/stress_spool.sh
stress_ng_available() {
  command -v stress-ng > /dev/null 2>&1
}

start_stress_ng_spool() {
  local dir="$1"
  local duration_sec="${2:-30}"
  local log_file="${3:-/tmp/ad-event-processor-stress-spool.log}"
  if ! stress_ng_available; then
    return 1
  fi
  [[ -d "$dir" ]] || return 1
  stress-ng \
    --hdd 1 \
    --hdd-bytes 64M \
    --hdd-method read \
    --timeout "${duration_sec}" \
    --temp-path "$dir" \
    >> "$log_file" 2>&1 &
  echo $!
}

stop_stress_ng_pid() {
  local pid="${1:-}"
  [[ -n "$pid" ]] || return 0
  kill "$pid" 2> /dev/null || true
  wait "$pid" 2> /dev/null || true
}

resolve_processor_ch_spool_mount() {
  local cid="$1"
  [[ -n "$cid" ]] || return 1
  docker inspect -f '{{range .Mounts}}{{if eq .Destination "/var/spool/ad-event-processor/ch"}}{{.Source}}{{end}}{{end}}' "$cid"
}

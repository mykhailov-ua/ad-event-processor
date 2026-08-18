#!/usr/bin/env bash
# stress-ng I/O pressure on a directory (CH spool fault drills).
set -euo pipefail

stress_ng_available() {
  command -v stress-ng > /dev/null 2>&1
}

start_stress_ng_spool() {
  local dir="$1"
  local duration_sec="${2:-30}"
  local log_file="${3:-/tmp/espx-stress-spool.log}"
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
  docker inspect -f '{{range .Mounts}}{{if eq .Destination "/var/spool/espx/ch"}}{{.Source}}{{end}}{{end}}' "$cid"
}

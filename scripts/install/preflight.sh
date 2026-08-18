#!/usr/bin/env bash
# Host preflight for appliance install (no Go required). Prefer bin/ad-event-processor-install when present.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/installer_env.sh"
cd "$ROOT"

STRICT=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --strict)
      STRICT=1
      shift
      ;;
    -h | --help)
      echo "usage: $0 [--strict]" >&2
      exit 0
      ;;
    *)
      echo "preflight: unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

if [[ -x "$ROOT/bin/ad-event-processor-install" ]]; then
  exec "$ROOT/bin/ad-event-processor-install" preflight $([[ "$STRICT" == "1" ]] && echo --strict)
fi

warn() { echo "preflight [warn] $*"; }
fail() { echo "preflight [fail] $*" >&2; }
pass() { echo "preflight [ok] $*"; }

kernel_check() {
  local ver major minor
  ver="$(uname -r 2> /dev/null | cut -d- -f1 || echo 0)"
  major="${ver%%.*}"
  minor="${ver#*.}"
  minor="${minor%%.*}"
  if [[ "$major" -lt 6 ]] || { [[ "$major" -eq 6 ]] && [[ "$minor" -lt 1 ]]; }; then
    warn "kernel ${ver} — appliance OK; edge XDP needs ≥ 6.1 + BTF"
  else
    pass "kernel ${ver}"
  fi
  if [[ ! -f /sys/kernel/btf/vmlinux ]]; then
    warn "BTF missing — required only for edge_xdp profile"
  fi
}

port_busy() {
  local port="$1"
  if command -v ss > /dev/null 2>&1; then
    ss -ltn "sport = :${port}" 2> /dev/null | grep -q ":${port}" && return 0
  fi
  return 1
}

ports_check() {
  local mgmt tracker db
  mgmt="$(installer_read_env MANAGEMENT_PORT)"
  mgmt="${mgmt:-8188}"
  tracker="$(installer_read_env TRACKER_PORT)"
  tracker="${tracker:-8181}"
  db="$(installer_read_env DB_PORT)"
  db="${db:-5430}"
  local p failed=0
  for p in "$mgmt" "$tracker" "$db"; do
    if port_busy "$p"; then
      warn "port ${p} appears in use (may be a previous install)"
    fi
  done
  pass "ports checked (management=${mgmt}, tracker=${tracker}, db=${db})"
}

mem_check() {
  local kb
  kb="$(awk '/MemTotal:/ {print $2}' /proc/meminfo 2> /dev/null || echo 0)"
  if [[ "$kb" -gt 0 ]] && [[ "$kb" -lt 7000000 ]]; then
    warn "RAM < 8 GB — appliance may be tight under load"
  else
    pass "memory"
  fi
}

docker_check() {
  if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then
    pass "docker"
    return 0
  fi
  warn "docker not ready — install will run provision (apt packages) unless --skip-provision"
  if [[ "$STRICT" == "1" ]]; then
    fail "docker required in --strict mode"
    return 1
  fi
  return 0
}

kernel_check
mem_check
ports_check
docker_check || exit 1
echo "preflight: done"

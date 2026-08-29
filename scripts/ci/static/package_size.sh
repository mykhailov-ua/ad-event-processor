#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

allowlist="${SCRIPTS}/ci/baselines/package_size_allowlist.txt"
max_files=40
max_bridge_lines=200
fail=0

is_allowlisted() {
  local key="$1"
  [[ -f "$allowlist" ]] && grep -qxF "$key" "$allowlist"
}

count_root_prod_go() {
  local pkg="$1"
  find "$pkg" -maxdepth 1 -name '*.go' ! -name '*_test.go' \
    ! -name '*_bpfel.go' ! -name '*_bpfeb.go' 2> /dev/null | wc -l | tr -d ' '
}

echo "package_size: root prod file count <= ${max_files}..."
for pkg in internal/* pkg/*; do
  [[ -d "$pkg" ]] || continue
  [[ "$pkg" == internal/db ]] && continue
  [[ "$pkg" == internal/pb ]] && continue
  n="$(count_root_prod_go "$pkg")"
  if ((n > max_files)); then
    if is_allowlisted "$pkg"; then
      echo "allowlisted ${pkg}: ${n} prod files"
    else
      echo "${pkg}: ${n} prod files (max ${max_files})" >&2
      fail=1
    fi
  fi
done

echo "package_size: controlplane bridge files <= ${max_bridge_lines} lines..."
for bridge in internal/controlplane/*_bridge.go; do
  [[ -f "$bridge" ]] || continue
  lines="$(wc -l < "$bridge" | tr -d ' ')"
  if ((lines > max_bridge_lines)); then
    key="bridge:${bridge}"
    if is_allowlisted "$key"; then
      echo "allowlisted ${bridge}: ${lines} lines"
    else
      echo "${bridge}: ${lines} lines (max ${max_bridge_lines})" >&2
      fail=1
    fi
  fi
done

if [[ "$fail" -ne 0 ]]; then
  echo "package_size_gate: FAILED" >&2
  exit 1
fi

echo "package_size_gate: OK"

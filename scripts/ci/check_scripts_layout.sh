#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail=0

workflow_refs=()
while IFS= read -r line; do
  [[ -n "$line" ]] || continue
  workflow_refs+=("$line")
done < <(grep -rhoE 'bash scripts/[a-zA-Z0-9_./-]+\.sh' .github/workflows Makefile Taskfile.yaml 2> /dev/null \
  | sed 's/^bash //' \
  | sort -u)

for path in "${workflow_refs[@]}"; do
  if [[ ! -f "$ROOT/$path" ]]; then
    echo "scripts-layout: CI/Makefile references missing $path" >&2
    fail=1
  fi
done

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

bash "$SCRIPTS/test/verify_redis_topology_test.sh"

echo "scripts-layout: OK (${#workflow_refs[@]} CI refs, redis topology test)"

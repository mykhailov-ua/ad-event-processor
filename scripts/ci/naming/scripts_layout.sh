#!/usr/bin/env bash
set -euo pipefail

# Role: Naming gate: Required scripts and Makefile refs.
# Execution context: CI via pr_fast or full_test.
# Invariants/contracts enforced: Legacy tokens and layout violations fail closed.
# Verify: bash scripts/ci/naming/scripts_layout.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
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

# Aggregate fail flag: exit 1 after all checks
if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

bash "$SCRIPTS/test/redis/verify_topology_test.sh"
bash "$SCRIPTS/test/redis/topology_lib_test.sh"

max_flat_leaves=20
allowlist_file="$SCRIPTS/ci/baselines/scripts_dir_ceiling_allowlist.txt"
declare -A allowlisted=()
if [[ -f "$allowlist_file" ]]; then
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" == \#* ]] && continue
    allowlisted["$line"]=1
  done < "$allowlist_file"
fi

for top in ci test fault lib ops perf security dev install lab clean tools; do
  dir="$SCRIPTS/$top"
  [[ -d "$dir" ]] || continue
  count=0
  while IFS= read -r -d '' f; do
    count=$((count + 1))
  done < <(find "$dir" -maxdepth 1 -type f \( -name '*.sh' -o -name '*.py' \) -print0 2> /dev/null)
  if [[ "$count" -gt "$max_flat_leaves" ]]; then
    key="scripts/$top"
    if [[ -n "${allowlisted[$key]:-}" ]]; then
      echo "scripts-layout: WARN $key has $count flat leaves (allowlisted; ceiling $max_flat_leaves)"
      continue
    fi
    echo "scripts-layout: $key has $count flat .sh/.py leaves (ceiling $max_flat_leaves)" >&2
    fail=1
  fi
done

# Aggregate fail flag: exit 1 after all checks
if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "scripts-layout: OK (${#workflow_refs[@]} CI refs, redis topology test, dir ceiling)"

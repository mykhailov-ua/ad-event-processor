#!/usr/bin/env bash
set -euo pipefail

# Role: Naming gate: State shard control file consistency.
# Execution context: CI via pr_fast or full_test.
# Invariants/contracts enforced: Legacy tokens and layout violations fail closed.
# Verify: bash scripts/ci/naming/state_shard_control.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

patterns=(
  'internal/controlplane/.*\.go'
  'internal/identity/.*\.go'
)

hits=()
for pattern in "${patterns[@]}"; do
  while IFS= read -r line; do
    hits+=("$line")
  done < <(rg -n 'rdbs\[0\]' "$ROOT/$pattern" 2> /dev/null \
    | rg -v '_test\.go' \
    | rg -v 'getRDB\(' \
    || true)
done

if ((${#hits[@]} > 0)); then
  echo "check_no_state_shard_control: forbidden rdbs[0] in control-plane paths:"
  printf '  %s\n' "${hits[@]}"
  exit 1
fi

echo "check_no_state_shard_control: ok"

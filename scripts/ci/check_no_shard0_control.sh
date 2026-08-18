#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
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
  echo "check_no_shard0_control: forbidden rdbs[0] in control-plane paths:"
  printf '  %s\n' "${hits[@]}"
  exit 1
fi

echo "check_no_shard0_control: ok"

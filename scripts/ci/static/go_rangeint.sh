#!/usr/bin/env bash
set -euo pipefail

# Role: Static gate: rangeint modernize static companion.
# Execution context: CI merge-pr-fast via pr_fast unless noted.
# Invariants/contracts enforced: Non-zero exit on contract violation; no silent pass on failure.
# Verify: bash scripts/ci/static/go_rangeint.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

pattern='for ([a-zA-Z_][a-zA-Z0-9_]*) := 0; \1 < [^;]+; \1\+\+'
allowlist=(
  internal/openrtb/parse26_scan.go:115
  internal/filter/netintel/residential_proxy.go:314
  internal/filter/netintel/residential_proxy.go:317
  internal/track/ip_rotation.go:84
  internal/dashboardadmin/service_role.go:156
  internal/ingest/parser/attestation.go:244
  internal/stream/fraud/fraud_stream_aggregate.go:139
)

is_allowlisted() {
  local hit="$1"
  local entry
  for entry in "${allowlist[@]}"; do
    [[ "$hit" == "$entry:"* ]] && return 0
  done
  return 1
}

mapfile -t hits < <(
  rg -n --pcre2 "$pattern" \
    --glob '*.go' \
    --glob '!**/pb/**' \
    --glob '!**/db/**' \
    --glob '!*_test.go' \
    internal/ pkg/ cmd/ 2> /dev/null || true
)

if ((${#hits[@]} == 0)); then
  echo "go_rangeint_gate: OK"
  exit 0
fi

filtered=()
for hit in "${hits[@]}"; do
  is_allowlisted "$hit" && continue
  filtered+=("$hit")
done

if ((${#filtered[@]} == 0)); then
  echo "go_rangeint_gate: OK"
  exit 0
fi

echo "go_rangeint_gate: use Go 1.22+ integer range (for i := range n) instead of 3-clause loops:" >&2
printf '%s\n' "${filtered[@]}" >&2
echo "go_rangeint_gate: analyzer rangeint (golang.org/x/tools/go/analysis/passes/modernize)" >&2
exit 1

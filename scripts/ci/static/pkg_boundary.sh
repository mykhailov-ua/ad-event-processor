#!/usr/bin/env bash
set -euo pipefail

# Role: Static gate: Import boundary matrix for internal packages.
# Execution context: CI merge-pr-fast via pr_fast unless noted.
# Invariants/contracts enforced: Non-zero exit on contract violation; no silent pass on failure.
# Verify: bash scripts/ci/static/pkg_boundary.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail=0

echo "pkg_boundary: production pkg/* must not import internal/*..."
if hits="$(rg -n 'ad-event-processor/internal' pkg/ --glob '*.go' --glob '!*_test.go' 2> /dev/null || true)"; then
  if [[ -n "$hits" ]]; then
    echo "$hits" >&2
    fail=1
  fi
fi

echo "pkg_boundary: daemon trees must live in internal/ only..."
for stale in pkg/broker/server pkg/regionproxy/server; do
  if [[ -d "$stale" ]]; then
    echo "forbidden directory: $stale" >&2
    fail=1
  fi
done

echo "pkg_boundary: merged single-consumer packages must stay deleted..."
for merged in \
  pkg/gtax \
  pkg/cpuset \
  pkg/runtimeautotune \
  pkg/vendorprobe \
  pkg/campaignmacro \
  pkg/bandit \
  pkg/doctor \
  pkg/pgfailover; do
  if [[ -d "$merged" ]]; then
    echo "merged package reappeared: $merged" >&2
    fail=1
  fi
done

echo "pkg_boundary: Tier C split roots must exist..."
for required in pkg/broker pkg/regionproxy; do
  if [[ ! -d "$required" ]]; then
    echo "missing Tier C package: $required" >&2
    fail=1
  fi
done

# Aggregate fail flag: exit 1 after all checks
if [[ "$fail" -ne 0 ]]; then
  echo "pkg_boundary_gate: FAILED" >&2
  exit 1
fi

echo "pkg_boundary_gate: OK"

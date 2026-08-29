#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail=0
checked=0

infra_pattern='testutil\.(SetupPostgres|SetupAdsPostgres|SetupAdsPaymentPostgres|SetupRedis)|setupTestDB|setupTestRedis|setupAuthTestInfra|setupAdsFaultInfra|setupClickHouseIntegration|setupClickHouseTest|setupClickHouseCostSyncTest|setupCostSyncDB|setupCHJanitorIntegration|database\.SetupTestDB|database\.SetupTestRedis|dbtest\.SetupTestDB|dbtest\.SetupTestRedis'
short_pattern='testing\.Short\(\)|setup[A-Za-z0-9_]*(Integration|Infra|TestDB|TestRedis|Test)\(t\)'
assert_pattern='require\.(NoError|Error|ErrorIs|Equal|NotEqual|NotEmpty|Empty|Len|True|False|Nil|NotNil)|assert\.(Equal|NotEqual|True|False|Len|NotEmpty|Empty|Error|ErrorIs)'

while IFS= read -r -d '' file; do
  checked=$((checked + 1))

  if ! rg -q "$short_pattern" "$file"; then
    echo "integration-test-slop: missing integration short guard in $file" >&2
    fail=1
    continue
  fi

  if ! rg -qi 't\.Skip\([^)]*integration' "$file"; then

    pkg_dir="$(dirname "$file")"
    if ! rg -qi 't\.Skip\([^)]*integration' "$pkg_dir" --glob '*_test.go' 2> /dev/null; then
      echo "integration-test-slop: skip reason must mention integration in $file (or package helpers)" >&2
      fail=1
    fi
  fi

  if rg -q 't\.Skip\(\)\s*$' "$file"; then
    echo "integration-test-slop: bare t.Skip() in $file" >&2
    fail=1
  fi

  if rg -q 'testify/mock|go\.uber\.org/mock|gomock\.' "$file"; then
    echo "integration-test-slop: mocks forbidden in integration tests: $file" >&2
    fail=1
  fi

  if ! rg -q "$infra_pattern" "$file"; then
    echo "integration-test-slop: missing real infra setup helper in $file" >&2
    fail=1
  fi

  if ! rg -q "$assert_pattern" "$file"; then
    echo "integration-test-slop: missing behavioral assertion in $file" >&2
    fail=1
  fi

  if rg -q 'scaffold_test: add a held-out negative assertion' "$file"; then
    echo "integration-test-slop: unresolved scaffold placeholder in $file" >&2
    fail=1
  fi

  if rg -q 'require\.NotNil\(t, pool, "integration scaffold must exercise real Postgres"\)' "$file"; then
    echo "integration-test-slop: placeholder Postgres-only assertion in $file" >&2
    fail=1
  fi
done < <(find internal pkg -type f -name '*integration_test.go' -print0 2> /dev/null || true)

if [[ "$fail" -ne 0 ]]; then
  echo "integration-test-slop: FAILED ($checked files checked)" >&2
  exit 1
fi

echo "integration-test-slop: OK ($checked files)"

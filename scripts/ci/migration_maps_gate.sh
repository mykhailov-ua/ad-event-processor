#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail=0

required=(
  deploy/vendor/migration/README.md
  deploy/vendor/migration/keitaro_macros.yaml
  deploy/vendor/migration/keitaro_sources.yaml
  deploy/vendor/migration/binom_macros.yaml
  deploy/vendor/migration/binom_sources.yaml
)

for path in "${required[@]}"; do
  if [[ ! -f "$path" ]]; then
    echo "migration_maps_gate: missing $path" >&2
    fail=1
  fi
done

if go test ./internal/migrationsource/ -run 'TestMigrationMaps|TestPreviewKeitaro|TestPreviewBinom|TestParseKeitaro|TestParseBinom|holdout' -count=1; then
  echo "migration_maps_gate: OK maps load and preview"
else
  echo "migration_maps_gate: FAIL migrationsource tests" >&2
  fail=1
fi

exit "$fail"

#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if ! go test ./internal/traffictemplates/... -count=1; then
  echo "traffic_source_templates_gate: FAIL traffictemplates tests" >&2
  exit 1
fi

if ! go run ./cmd/codegen-traffic-templates --check; then
  echo "traffic_source_templates_gate: FAIL stale generated TS" >&2
  exit 1
fi

echo "traffic_source_templates_gate: OK"

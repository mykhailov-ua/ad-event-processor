#!/usr/bin/env bash
set -euo pipefail

# Role: Prometheus rules validation.
# Execution context: CI via tier_a tail.
# Invariants/contracts enforced: Invalid rules fail.
# Verify: bash scripts/ci/prometheus_rules_check.sh
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

RULES="$ROOT/deploy/monitoring/prometheus.rules.yaml"
[[ -f "$RULES" ]] || {
  echo "prometheus_rules_check: missing $RULES" >&2
  exit 1
}

go test ./deploy/monitoring/ -count=1

if command -v promtool > /dev/null 2>&1; then
  promtool check rules "$RULES"
  echo "prometheus_rules_check: promtool ok"
else
  echo "prometheus_rules_check: promtool not installed - go test only"
fi

echo "prometheus_rules_check: ok"

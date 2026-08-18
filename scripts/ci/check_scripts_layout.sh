#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

required=(
  scripts/lib/paths.sh
  scripts/lib/safe_paths.sh
  scripts/ci/deps.sh
  scripts/ci/management_domain_coverage.sh
  scripts/ci/validate_configs.sh
  scripts/ci/pr_fast.sh
  scripts/ci/admin_web.sh
  scripts/ci/admin_release_gate.sh
  scripts/ci/confirm_registry_audit.sh
  scripts/dev/preflight.sh
  scripts/dev/stack.sh
  scripts/dev/smoke_ingest_only.sh
  scripts/dev/smoke_network_operator.sh
  scripts/dev/smoke_analytics_ml.sh
  scripts/test/gate_run.sh
  scripts/ops/phase0.sh
  scripts/ops/verify_redis_topology.sh
  scripts/test/run_resilience.sh
  scripts/test/test_resilience.sh
  scripts/fault/run.sh
  scripts/fault/test_resilience.sh
  scripts/fault/sentinel_failover_env.sh
  scripts/fault/sentinel.sh
)

fail=0
for path in "${required[@]}"; do
  if [[ ! -f "$ROOT/$path" ]]; then
    echo "scripts-layout: missing $path" >&2
    fail=1
  fi
done

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "scripts-layout: OK (${#required[@]} paths)"

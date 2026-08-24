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
  scripts/ci/check_repo_clutter.sh
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
  scripts/ops/edge_preflight.sh
  scripts/ops/verify_redis_topology.sh
  scripts/test/run_resilience.sh
  scripts/fault/run.sh
  scripts/fault/sentinel_failover_env.sh
  scripts/test/sentinel.sh
  scripts/test/nginx_lua_tests.sh
  scripts/test/cpa_compliance_smoke.sh
  scripts/test/verify_redis_topology_test.sh
)

fail=0
for path in "${required[@]}"; do
  if [[ ! -f "$ROOT/$path" ]]; then
    echo "scripts-layout: missing $path" >&2
    fail=1
  fi
done

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

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

bash "$SCRIPTS/test/verify_redis_topology_test.sh"

echo "scripts-layout: OK (${#required[@]} required, ${#workflow_refs[@]} CI refs, redis topology test)"

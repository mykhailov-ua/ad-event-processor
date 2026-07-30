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
	scripts/dev/preflight.sh
	scripts/dev/stack.sh
	scripts/perf/gate_run.sh
	scripts/edge/phase0.sh
	scripts/local-dev/dev_preflight.sh
	scripts/local-dev/smoke_ingest_only.sh
	scripts/local-dev/smoke_network_operator.sh
	scripts/local-dev/smoke_analytics_ml.sh
	scripts/perf-gate/perf_gate_run.sh
	scripts/edge-tuning/edge_phase0.sh
	scripts/redis/verify_topology.sh
	scripts/deploy/verify_redis_topology.sh
	scripts/fault/run.sh
	scripts/fault/test_resilience.sh
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

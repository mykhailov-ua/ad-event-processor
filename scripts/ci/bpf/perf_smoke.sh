#!/usr/bin/env bash
set -euo pipefail

# Role: BPF gate: BPF perf smoke on PR runners.
# Execution context: Perf runner or nightly; resource.sh skips on github-hosted without PERF_RUNNER_LABEL.
# Invariants/contracts enforced: Strict BPF thresholds from load-test-bpf.mdc when enabled.
# Verify: bash scripts/ci/bpf/perf_smoke.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

export PERF_GATE_STRICT=false
bash "$SCRIPTS/test/load/gate_run.sh"

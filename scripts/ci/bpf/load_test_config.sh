#!/usr/bin/env bash
set -euo pipefail

# Role: BPF gate: Load-test config validation for BPF.
# Execution context: Perf runner or nightly; resource.sh skips on github-hosted without PERF_RUNNER_LABEL.
# Invariants/contracts enforced: Strict BPF thresholds from load-test-bpf.mdc when enabled.
# Verify: bash scripts/ci/bpf/load_test_config.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"

bash "$SCRIPTS/lib/render_load_test_config.sh" --check
echo "load_test_config_gate: ok"

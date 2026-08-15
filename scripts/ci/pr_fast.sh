#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

bash "$SCRIPTS/ci/validate_configs.sh"
bash "$SCRIPTS/ci/tier_a.sh"
bash "$SCRIPTS/ci/check_scripts_layout.sh"
bash "$SCRIPTS/ci/compliance.sh"
bash "$SCRIPTS/ci/ch_direct.sh"
bash "$SCRIPTS/ci/lint_go_gate.sh" all
bash "$SCRIPTS/ci/integration_skip_reason_gate.sh"
make test-alloc-gate
make test-fast
bash "$SCRIPTS/ci/shard0_nil_gate.sh"
bash "$SCRIPTS/ci/cold_path_json_gate.sh"
bash "$SCRIPTS/ci/capi_staging_gate.sh"
bash "$SCRIPTS/ci/check_no_espx.sh"
bash "$SCRIPTS/ci/check_no_espx_core.sh"
bash "$SCRIPTS/ci/admin_web.sh"

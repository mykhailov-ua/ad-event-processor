#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

bash "$SCRIPTS/ci/validate_configs.sh"
bash "$SCRIPTS/ci/load_test_config_gate.sh"
bash "$SCRIPTS/ci/check_repo_clutter.sh"
bash "$SCRIPTS/ci/tier_a.sh"
bash "$SCRIPTS/ci/check_scripts_layout.sh"
bash "$SCRIPTS/ci/compliance.sh"
bash "$SCRIPTS/ci/ch_direct.sh"
if [[ "${SKIP_LINT:-}" != "1" ]]; then
  bash "$SCRIPTS/ci/lint_gate.sh"
fi
bash "$SCRIPTS/ci/integration_test_slop_gate.sh"
bash "$SCRIPTS/ci/anti_slop_gate.sh"
bash "$SCRIPTS/ci/diff_assertion_gate.sh"
bash "$SCRIPTS/ci/sql_safety_gate.sh"
bash "$SCRIPTS/ci/hot_path_static_gate.sh"
bash "$SCRIPTS/ci/cold_path_static_gate.sh"
bash "$SCRIPTS/ci/escape_heap_gate.sh"
make test-alloc-gate
make test-fast
bash "$SCRIPTS/ci/shard0_nil_gate.sh"
bash "$SCRIPTS/ci/cold_path_json_gate.sh"
bash "$SCRIPTS/ci/capi_staging_gate.sh"
bash "$SCRIPTS/ci/migration_maps_gate.sh"
bash "$SCRIPTS/ci/ingestion_migration_version_gate.sh"
bash "$SCRIPTS/ci/traffic_source_templates_gate.sh"
bash "$SCRIPTS/ci/check_no_legacy_naming.sh"
bash "$SCRIPTS/ci/antifraud_doc_gate.sh"
bash "$SCRIPTS/ci/admin_web.sh"

#!/usr/bin/env bash
set -euo pipefail

# Role: PR-fast merge orchestrator; composes static gates, alloc gate, and test-fast once.
# Execution context: CI merge-pr-fast job and local pre-PR; not a full integration/fault tier.
# Invariants/contracts enforced: Each child gate runs at most once; SKIP_LINT=1 skips lint because merge-lint owns it.
# Verify: bash scripts/ci/pr_fast.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

bash "$SCRIPTS/ci/validate_configs.sh"
bash "$SCRIPTS/ci/bpf/load_test_config.sh"
bash "$SCRIPTS/ci/naming/repo_clutter.sh"
bash "$SCRIPTS/ci/tier_a.sh"
bash "$SCRIPTS/ci/static/doc_go_verify.sh"
bash "$SCRIPTS/ci/static/pkg_boundary.sh"
bash "$SCRIPTS/ci/static/package_size.sh"
bash "$SCRIPTS/ci/naming/scripts_layout.sh"
bash "$SCRIPTS/ci/compliance.sh"
bash "$SCRIPTS/ci/ch_direct.sh"
# merge-lint owns lint; pr_fast skips when SKIP_LINT=1
if [[ "${SKIP_LINT:-}" != "1" ]]; then
  bash "$SCRIPTS/ci/lint.sh"
fi
bash "$SCRIPTS/ci/static/integration_test_slop.sh"
bash "$SCRIPTS/ci/static/anti_slop.sh"
bash "$SCRIPTS/ci/static/diff_assertion.sh"
bash "$SCRIPTS/ci/static/sql_safety.sh"
bash "$SCRIPTS/ci/static/hot_path_static.sh"
bash "$SCRIPTS/ci/static/cold_path_static.sh"
bash "$SCRIPTS/ci/static/escape_heap.sh"
make test-alloc-gate
make test-fast
bash "$SCRIPTS/ci/static/state_shard_nil.sh"
bash "$SCRIPTS/ci/static/cold_path_json.sh"
bash "$SCRIPTS/ci/static/capi_staging.sh"
bash "$SCRIPTS/ci/static/migration_maps.sh"
bash "$SCRIPTS/ci/static/ingestion_migration_version.sh"
bash "$SCRIPTS/ci/static/traffic_source_templates.sh"
bash "$SCRIPTS/ci/naming/legacy_naming.sh"
bash "$SCRIPTS/ci/naming/antifraud_doc.sh"
bash "$SCRIPTS/ci/admin/web.sh"

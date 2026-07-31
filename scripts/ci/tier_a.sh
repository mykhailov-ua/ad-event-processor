#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

export FIND_OBVIOUS_COMMENTS_FAIL=1

bash "$SCRIPTS/ci/check_docs_layout.sh"
bash "$SCRIPTS/ci/check_no_milestone_refs.sh"
bash "$SCRIPTS/ci/check_no_html_success.sh"
bash "$SCRIPTS/ci/check_no_service_slog.sh"
bash "$SCRIPTS/ci/check_error_handling.sh"
bash "$SCRIPTS/ci/check_brand_boundary.sh"
bash "$SCRIPTS/ci/find_obvious_comments.sh"

echo "tier_a: OK"

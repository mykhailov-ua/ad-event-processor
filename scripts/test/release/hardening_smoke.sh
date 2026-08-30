#!/usr/bin/env bash
# Role: Release hardening smoke for garble policy and ingest garble holdout tests.
# Execution context: Repo root; invokes scripts/ci/license/garble_literals_policy.sh.
# Env knobs: none.
# Verify: bash scripts/test/release/hardening_smoke.sh
set -euo pipefail
cd "$(dirname "$0")/../../.."
bash scripts/ci/license/garble_literals_policy.sh
go test ./internal/ingest/ -run '^TestGarblePolicy_' -count=1
echo "release_hardening_smoke: OK"

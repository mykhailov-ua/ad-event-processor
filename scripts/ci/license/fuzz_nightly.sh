#!/usr/bin/env bash

set -euo pipefail

# Role: License gate: License JWT fuzz nightly smoke.
# Execution context: CI license-verify tier or release QA.
# Invariants/contracts enforced: Required rows fail closed; optional rows use skip_gate with env flags.
# Verify: bash scripts/ci/license/fuzz_nightly.sh
cd "$(dirname "$0")/../../.."
go test ./internal/licensing/ -run '^TestReleaseQA_' -count=1
echo "license_fuzz_nightly_gate: OK"

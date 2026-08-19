#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")/../.."
bash scripts/ci/garble_literals_policy_gate.sh
go test ./internal/ingestion/ -run '^TestGarblePolicy_' -count=1
echo "release_hardening_smoke: OK"

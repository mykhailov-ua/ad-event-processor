#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")/../../.."
bash scripts/ci/license/garble_literals_policy.sh
go test ./internal/ingest/ -run '^TestGarblePolicy_' -count=1
echo "release_hardening_smoke: OK"

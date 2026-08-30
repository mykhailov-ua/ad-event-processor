#!/usr/bin/env bash
# Role: Pre-tag release QA fuzz smoke and garbled alloc gate with fault_proof telemetry.
# Execution context: Repo root; timeboxed go fuzz on licensing package.
# Env knobs: none (fuzztime fixed in script: 10s/5s).
# Verify: bash scripts/test/release/qa_smoke.sh
set -euo pipefail
cd "$(dirname "$0")/../../.."

echo "release_qa_smoke: fuzz smoke (VerifyJWT 10s, DecodeUnverified/JSONClaims 5s each)"
go test ./internal/licensing/ -fuzz=FuzzVerifyJWT -fuzztime=10s -count=1
go test ./internal/licensing/ -fuzz=FuzzDecodeUnverified -fuzztime=5s -count=1
go test ./internal/licensing/ -fuzz=FuzzJSONClaims -fuzztime=5s -count=1

bash scripts/ci/license/garbled_alloc.sh

# fault_proof telemetry for release QA tier (grep in CI resilience logs).
echo "fault_proof fault=release_qa_smoke harness=release_qa_fuzz_smoke pass=1"
echo "release_qa_smoke: OK"

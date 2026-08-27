#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")/../.."

echo "release_qa_smoke: fuzz smoke (VerifyJWT 10s, DecodeUnverified/JSONClaims 5s each)"
go test ./internal/licensing/ -fuzz=FuzzVerifyJWT -fuzztime=10s -count=1
go test ./internal/licensing/ -fuzz=FuzzDecodeUnverified -fuzztime=5s -count=1
go test ./internal/licensing/ -fuzz=FuzzJSONClaims -fuzztime=5s -count=1

bash scripts/ci/license_garbled_alloc_gate.sh

echo "fault_proof fault=release_qa_smoke harness=release_qa_fuzz_smoke pass=1"
echo "release_qa_smoke: OK"

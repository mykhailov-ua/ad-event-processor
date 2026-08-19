#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")/../.."
go test ./internal/licensing/ -run '^TestReleaseQA_' -count=1
echo "license_fuzz_nightly_gate: OK"
